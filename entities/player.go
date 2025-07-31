package entities

import (
	"sync"

	"github.com/AmirRezaM75/kenopsiarelay/pkg/logx"
	"github.com/gorilla/websocket"

	"go.uber.org/zap"
)

type Player struct {
	Id     string
	GameId string
	Index  int
	// User data
	Username    string
	AvatarId    uint8
	IsBot       bool
	IsConnected bool
	// To keep track of closed channel
	IsClosed   bool
	Connection *websocket.Conn
	Message    chan []byte
	mutex      sync.Mutex
}

// Different scenarios for 'close of closed channel'
// 1) If user opens duplicate tab and close the first one

func (player *Player) Kick() {
	// We are using mutex to make sure IsClosed value is evaluated correctly
	// when reading its value at the same time.
	// https://go101.org/article/channel-closing.html
	player.mutex.Lock()

	defer player.mutex.Unlock()

	if !player.IsClosed {
		close(player.Message)
		player.IsClosed = true
	}

	// First we need to check if it's nil or not
	// we call kick method in game_handler, and player may have no connection
	if player.Connection != nil {
		err := player.Connection.Close()

		if err != nil {
			logx.Logger.Info(
				err.Error(),
				zap.String("desc", "could not close player connection"),
				zap.String("playerId", player.Id),
			)
		}
	}

	player.IsConnected = false
}

// Reconnect safely handles player reconnection with proper mutex protection
// This method prevents race conditions during player reconnection
// by atomically updating all player state under mutex protection
func (player *Player) Reconnect(connection *websocket.Conn) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	// We don't use Kick() here because it would cause double-locking of the mutex
	if !player.IsClosed {
		close(player.Message)
		player.IsClosed = true
	}

	if player.Connection != nil {
		_ = player.Connection.Close()
	}

	player.Message = make(chan []byte, 50)
	player.IsClosed = false
	player.Connection = connection
	player.IsConnected = true
}

func (player *Player) Write() {
	defer player.Kick()

	for {
		message, ok := <-player.Message

		if !ok {
			logx.Logger.Info(
				"player channel is closed!",
				zap.String("playerId", player.Id),
			)
			return
		}

		err := player.Connection.WriteMessage(websocket.BinaryMessage, message)

		if err != nil {
			// Check if this is an unexpected connection error
			// Also handle "use of closed network connection" which is not a WebSocket close error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				logx.Logger.Error(
					err.Error(),
					zap.String("desc", "unexpected websocket write error"),
					zap.String("playerId", player.Id),
				)
			} else {
				logx.Logger.Info(
					err.Error(),
					zap.String("error", err.Error()),
					zap.String("desc", "expected websocket write error"),
					zap.String("playerId", player.Id),
				)
			}
			// Continuing the loop would cause infinite error logging if connection is broken
			// Better to exit and let the connection cleanup happen
			return
		}
	}
}

// unsubscribe is a generic function to unsubscribe a player from a hub
func unsubscribe[S GameState](player *Player, hub *Hub[S]) {
	if game := hub.FindGame(player.GameId); game != nil {
		err := hub.OnPlayerLeft(hub, game, player)

		if err != nil {
			logx.Logger.Error(
				err.Error(),
				zap.String("desc", "could not execute handler when player is left"),
				zap.String("gameId", game.Id),
				zap.String("playerId", player.Id),
			)
			return
		}
	}
}

func Read[S GameState](player *Player, hub *Hub[S]) {
	defer func() {
		player.Kick()
		unsubscribe(player, hub)
	}()

	for {
		// CONTEXT CANCELLATION FIX: Check if context is cancelled before each read attempt
		// This ensures the goroutine can exit gracefully when user shuts down the server
		// We use select with default to make this non-blocking
		select {
		case <-hub.Context.Done():
			// Context cancelled - shutdown gracefully
			return
		default:
			// Continue to read
		}

		_, message, err := player.Connection.ReadMessage()

		if err != nil {
			// This logs only unexpected errors, not normal browser refresh/close events
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				logx.Logger.Error(
					err.Error(),
					zap.String("desc", "unexpected websocket close error"),
					zap.String("playerId", player.Id),
				)
			} else {
				logx.Logger.Info(
					"WebSocket connection closed by client",
					zap.String("error", err.Error()),
					zap.String("desc", "client disconnected"),
					zap.String("playerId", player.Id),
				)
			}
			return
		}

		react(player, message, hub)
	}
}

// react is a generic function to handle player reactions
func react[S GameState](player *Player, message []byte, hub *Hub[S]) {
	game := hub.FindGame(player.GameId)

	if game == nil {
		return
	}

	err := hub.OnMessageReceived(hub, game, player, message)

	if err != nil {
		logx.Logger.Error(
			err.Error(),
			zap.String("desc", "could not handle incoming message"),
			zap.String("gameId", game.Id),
			zap.String("playerId", player.Id),
		)
		return
	}
}
