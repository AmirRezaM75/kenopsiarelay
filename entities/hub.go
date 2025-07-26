package entities

import (
	"github.com/amirrezam75/kenopsiarelay/pkg/syncx"
	"github.com/amirrezam75/kenopsiarelay/schemas"
)

type Hub struct {
	Games syncx.Map[string, *Game]
	// We must use a pointer for this type because protoimpl.MessageState
	// includes a sync.Mutex field. The sync.Mutex type implements the sync.Locker interface,
	// which means it is not safe to copy. If we were to use a value type instead of a pointer,
	// the sync.Mutex would be copied along with the struct, leading to potential race conditions
	// and unpredictable behavior due to multiple copies of the same mutex.
	// Using a pointer ensures that there is only one instance of the mutex,
	// maintaining proper synchronization across all operations.
	Dispatch chan *schemas.DispatcherMessage
	// MessageHandler is responsible for processing incoming messages from connected clients.
	// Within this handler, you should parse the incoming request, perform any necessary validation,
	// and update your game state accordingly based on the content of the message.
	// The message is provided as a raw []byte, so it is your responsibility to decode it.
	// You can choose the appropriate deserialization method, such as JSON unmarshalling or using
	// a binary protocol like Protobuf, depending on your application's design and performance needs.
	MessageHandler MessageHandler
}

// NewHub contains an entity that uses mutexes for synchronization,
// and passing locks by value is not a good practice.
// Therefore, all receivers are passed by pointer to avoid copying
// locks and ensure proper synchronization.
func NewHub(messageHandler MessageHandler) *Hub {
	return &Hub{
		Dispatch:       make(chan *schemas.DispatcherMessage, 500),
		MessageHandler: messageHandler,
	}
}

func (hub *Hub) Run() {
	for {
		select {
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

type MessageHandler func(hub *Hub, game *Game, player *Player, message []byte) error

func (hub *Hub) FindGame(id string) *Game {
	game, exists := hub.Games.Load(id)

	if !exists {
		return nil
	}

	return game
}

// TODO: Cleanup
