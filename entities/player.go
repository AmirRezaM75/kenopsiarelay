package entities

import (
	"sync"

	"github.com/amirrezam75/kenopsiarelay/pkg/logx"
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
			logx.Logger.Error(
				err.Error(),
				zap.String("desc", "could not close player connection"),
				zap.String("playerId", player.Id),
			)
		}
	}

	player.IsConnected = false
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
			break
		}

		err := player.Connection.WriteMessage(websocket.BinaryMessage, message)

		if err != nil {
			logx.Logger.Error(
				err.Error(),
				zap.String("desc", "could not write player message"),
				zap.String("playerId", player.Id),
			)
		}
	}
}

// unsubscribe is a generic function to unsubscribe a player from a hub
func unsubscribe[S GameState](player *Player, hub *Hub[S]) {
	if game := hub.FindGame(player.GameId); game != nil {
		//game.Left(hub, player.Id)
	}
}

// Read is a generic function to read messages for a player
func Read[S GameState](player *Player, hub *Hub[S]) {
	defer func() {
		player.Kick()
		unsubscribe(player, hub)
	}()

	for {
		_, message, err := player.Connection.ReadMessage()

		if err != nil {
			logx.Logger.Error(
				err.Error(),
				zap.String("desc", "could not read player message"),
				zap.String("playerId", player.Id),
			)
			break
		}

		// TODO: Unmarshal

		react(player, message, hub)
	}
}

// react is a generic function to handle player reactions
func react[S GameState](player *Player, message []byte, hub *Hub[S]) {
	game := hub.FindGame(player.GameId)

	if game == nil {
		return
	}

	err := hub.MessageHandler(hub, game, player, message)

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
