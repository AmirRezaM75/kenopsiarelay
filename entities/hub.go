package entities

import (
	"context"

	"github.com/AmirRezaM75/kenopsiarelay/pkg/syncx"
	"github.com/AmirRezaM75/kenopsiarelay/schemas"
)

type Hub[S GameState] struct {
	GameSlug string
	Games    syncx.Map[string, *Game[S]]

	Context context.Context

	// We must use a pointer for this type because protoimpl.MessageState
	// includes a sync.Mutex field. The sync.Mutex type implements the sync.Locker interface,
	// which means it is not safe to copy. If we were to use a value type instead of a pointer,
	// the sync.Mutex would be copied along with the struct, leading to potential race conditions
	// and unpredictable behavior due to multiple copies of the same mutex.
	// Using a pointer ensures that there is only one instance of the mutex,
	// maintaining proper synchronization across all operations.
	Dispatch chan *schemas.DispatcherMessage
	// OnMessageReceived is responsible for processing incoming messages from connected clients.
	// Within this handler, you should parse the incoming request, perform any necessary validation,
	// and update your game state accordingly based on the content of the message.
	// The message is provided as a raw []byte, so it is your responsibility to decode it.
	// You can choose the appropriate deserialization method, such as JSON unmarshalling or using
	// a binary protocol like Protobuf, depending on your application's design and performance needs.
	OnMessageReceived MessageReceivedHandler[S]
	OnPlayerJoined    PlayerJoinedHandler[S]
	OnPlayerLeft      PlayerLeftHandler[S]
	OnGameCreated     GameCreatedHandler[S]
	// GameStateFactory creates new game states
	GameStateFactory func() S
}

// NewHub creates a new hub with context for lifecycle management
// Accept context from user application level
// This follows Go best practices where context flows down from caller to callee
// The context controls when the hub should shut down gracefully
func NewHub[S GameState](
	ctx context.Context,
	dispatchBufferSize int,
	gameSlug string,
	messageReceivedHandler MessageReceivedHandler[S],
	playerJoinedHandler PlayerJoinedHandler[S],
	playerLeftHandler PlayerLeftHandler[S],
	gameStateFactory func() S,
) *Hub[S] {
	// Ensure buffer size is positive, use default if not specified
	// Zero or negative values could cause unbuffered channels or panics
	bufferSize := dispatchBufferSize

	if bufferSize <= 0 {
		bufferSize = 500
	}

	return &Hub[S]{
		GameSlug:          gameSlug,
		Context:           ctx,
		Dispatch:          make(chan *schemas.DispatcherMessage, bufferSize),
		OnMessageReceived: messageReceivedHandler,
		OnPlayerJoined:    playerJoinedHandler,
		OnPlayerLeft:      playerLeftHandler,
		GameStateFactory:  gameStateFactory,
	}
}

// Run starts the hub's message dispatch loop with user-controlled graceful shutdown
// When user cancels the context (e.g., on SIGTERM), hub shuts down gracefully
// This ensures all player connections are closed and resources are cleaned up
func (hub *Hub[S]) Run() {
	for {
		select {
		case <-hub.Context.Done():
			hub.Games.Range(func(gameId string, game *Game[S]) bool {
				game.Players.Range(func(playerId string, player *Player) bool {
					player.Kick()
					return true
				})
				return true
			})
			return
		case message := <-hub.Dispatch:
			if game := hub.FindGame(message.GameId); game != nil {
				for _, receiverId := range message.ReceiverIds {
					if player, ok := game.Players.Load(receiverId); ok {
						func() {
							player.mutex.Lock()
							defer player.mutex.Unlock()

							if !player.IsClosed {
								player.Message <- message.Body
							}
						}()
					}
				}
			}
		}
	}
}

type MessageReceivedHandler[S GameState] func(hub *Hub[S], game *Game[S], player *Player, message []byte) error
type PlayerJoinedHandler[S GameState] func(hub *Hub[S], game *Game[S], player *Player) error
type PlayerLeftHandler[S GameState] func(hub *Hub[S], game *Game[S], player *Player) error
type GameCreatedHandler[S GameState] func(hub *Hub[S], game *Game[S]) error

func (hub *Hub[S]) FindGame(id string) *Game[S] {
	game, exists := hub.Games.Load(id)

	if !exists {
		return nil
	}

	return game
}

// RemoveGame removes a game from the hub to prevent memory leaks
func (hub *Hub[S]) RemoveGame(gameId string) {
	if game, exists := hub.Games.Load(gameId); exists {
		game.Players.Range(func(playerId string, player *Player) bool {
			player.Kick()
			return true
		})
		hub.Games.Delete(gameId)
	}
}
