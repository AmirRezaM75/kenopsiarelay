package gameserver

import (
	"math/rand"
	"time"

	"github.com/AmirRezaM75/kenopsiarelay/entities"
	"github.com/AmirRezaM75/kenopsiarelay/handlers"
	"github.com/AmirRezaM75/kenopsiarelay/pkg/logx"
	"github.com/AmirRezaM75/kenopsiarelay/schemas"
	"github.com/AmirRezaM75/kenopsiarelay/services"
	"github.com/amirrezam75/kenopsiacommon/middlewares"
	"github.com/amirrezam75/kenopsialobby"
	"github.com/amirrezam75/kenopsiauser"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
)

// GameServer encapsulates all game server functionality
type GameServer[S entities.GameState] struct {
	router      *chi.Mux
	middlewares Middlewares
	hub         *entities.Hub[S]
}

type Middlewares struct {
	auth middlewares.Authenticate
}

// NewGameServer creates a new game server with the provided configuration
func NewGameServer[S entities.GameState](config Config[S]) *GameServer[S] {
	// Previously rand.Seed() was called on every game creation which could
	// lead to identical seeds for rapidly created games, reducing randomness
	rand.Seed(time.Now().UnixNano())

	logx.NewLogger()

	hub := entities.NewHub(
		config.Context,
		config.DispatchBufferSize,
		config.GameSlug,
		config.OnMessageReceived,
		config.OnPlayerJoined,
		config.OnPlayerLeft,
		config.GameStateFactory,
	)

	userRepository := kenopsiauser.NewUserRepository(
		config.UserService.BaseURL,
		config.UserService.Token,
	)

	lobbyRepository := kenopsialobby.NewLobbyRepository(
		config.LobbyService.BaseURL,
		config.LobbyService.Token,
	)

	publisherService := services.NewPublisherService(
		config.Publisher.Redis.Host,
		config.Publisher.Redis.Port,
		config.Publisher.Redis.Password,
	)

	gameService := services.NewGameService(hub, userRepository, lobbyRepository, publisherService)

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.Router.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authMiddleware := middlewares.NewAuthenticateMiddleware(userRepository)

	serviceAdapter := &gameServiceAdapter[S]{gameService: gameService}

	handlers.NewGameHandler(router, serviceAdapter, authMiddleware)

	gameServer := &GameServer[S]{
		router:      router,
		hub:         hub,
		middlewares: Middlewares{auth: authMiddleware},
	}

	go hub.Run()

	return gameServer
}

// GetRouter returns the configured router
func (gs *GameServer[S]) GetRouter() *chi.Mux {
	return gs.router
}

// GetHub returns the hub instance
func (gs *GameServer[S]) GetHub() *entities.Hub[S] {
	return gs.hub
}

func (gs *GameServer[S]) GetAuthMiddleware() middlewares.Authenticate {
	return gs.middlewares.auth
}

// Shutdown provides explicit shutdown method for immediate cleanup
// Note: Hub will also shut down automatically when user cancels the context
func (gs *GameServer[S]) Shutdown() {
	gs.hub.Games.Range(func(gameId string, game *entities.Game[S]) bool {
		game.Players.Range(func(playerId string, player *entities.Player) bool {
			player.Kick()
			return true
		})
		return true
	})
}

// gameServiceAdapter adapts the generic service to the expected interface
type gameServiceAdapter[S entities.GameState] struct {
	gameService services.GameService[S]
}

func (a *gameServiceAdapter[S]) Create(user kenopsiauser.User, payload schemas.CreateGameRequest) (*schemas.CreateGameResponse, error) {
	return a.gameService.Create(user, payload)
}

func (a *gameServiceAdapter[S]) Join(gameId, ticketId string, connection *websocket.Conn) (func(), error) {
	return a.gameService.Join(gameId, ticketId, connection)
}
