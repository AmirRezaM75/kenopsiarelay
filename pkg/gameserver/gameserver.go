package gameserver

import (
	middlwares "github.com/amirrezam75/kenopsiacommon/middlwares"
	"github.com/amirrezam75/kenopsialobby"
	"github.com/amirrezam75/kenopsiarelay/entities"
	"github.com/amirrezam75/kenopsiarelay/handlers"
	"github.com/amirrezam75/kenopsiarelay/pkg/logx"
	"github.com/amirrezam75/kenopsiarelay/schemas"
	"github.com/amirrezam75/kenopsiarelay/services"
	"github.com/amirrezam75/kenopsiauser"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
)

// GameServer encapsulates all game server functionality
type GameServer[S entities.GameState] struct {
	router *chi.Mux
}

// NewGameServer creates a new game server with the provided configuration
func NewGameServer[S entities.GameState](config Config[S]) *GameServer[S] {
	logx.NewLogger()

	hub := entities.NewHub(config.OnMessage, config.GameStateFactory)

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

	authMiddleware := middlwares.NewAuthenticateMiddleware(userRepository)

	serviceAdapter := &gameServiceAdapter[S]{gameService: gameService}

	handlers.NewGameHandler(router, serviceAdapter, authMiddleware)

	gameServer := &GameServer[S]{
		router: router,
	}

	go hub.Run()

	return gameServer
}

// GetRouter returns the configured router
func (gs *GameServer[S]) GetRouter() *chi.Mux {
	return gs.router
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
