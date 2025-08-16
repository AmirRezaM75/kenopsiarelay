package handlers

import (
	"errors"
	"net/http"
	"os"

	"github.com/AmirRezaM75/kenopsiarelay/pkg/logx"
	"github.com/AmirRezaM75/kenopsiarelay/schemas"
	"github.com/AmirRezaM75/kenopsiarelay/services"
	"github.com/amirrezam75/kenopsiacommon/middlewares"
	commonservices "github.com/amirrezam75/kenopsiacommon/services"
	"github.com/amirrezam75/kenopsiauser"
	"github.com/go-chi/chi/v5"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("origin") == os.Getenv("FRONTEND_URL") // TODO: Config
	},
}

// GameServiceInterface defines the operations needed by the handler
type GameServiceInterface interface {
	Create(user kenopsiauser.User, payload schemas.CreateGameRequest) (*schemas.CreateGameResponse, error)
	Join(gameId, ticketId string, connection *websocket.Conn) (func(), error)
}

type GameHandler struct {
	gameService GameServiceInterface
}

func NewGameHandler(
	router *chi.Mux,
	gameService GameServiceInterface,
	authMiddleware middlewares.Authenticate,
) {
	gameHandler := GameHandler{gameService: gameService}
	router.With(authMiddleware.Handle).Post("/games", gameHandler.create)
	router.Get("/games/{id}/join", gameHandler.join)
}

func (gameHandler GameHandler) create(w http.ResponseWriter, r *http.Request) {
	user := commonservices.ContextService{}.GetUser(r.Context())

	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var payload schemas.CreateGameRequest

	err := decode(&payload, r)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		encode(schemas.ErrorResponse{Message: "The given payload is invalid."}, w)
		logx.Logger.Warn(err.Error(), zap.String("desc", "could not decode CreateGameRequest"))
		return
	}

	response, err := gameHandler.gameService.Create(*user, payload)
	if err != nil {
		if errors.Is(err, services.LobbyNotFound) {
			w.WriteHeader(http.StatusNotFound)
			encode(schemas.ErrorResponse{Message: "Lobby not found."}, w)
			return
		}

		w.WriteHeader(http.StatusUnprocessableEntity)
		encode(schemas.ErrorResponse{Message: "Something goes wrong!"}, w)
		return
	}

	w.WriteHeader(http.StatusCreated)

	encode(response, w)
}

func (gameHandler GameHandler) join(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		logx.Logger.Error(
			err.Error(),
			zap.String("desc", "could not upgrade http request"),
		)
		w.WriteHeader(400)
		return
	}

	gameId := r.PathValue("id")

	ticketId := r.URL.Query().Get("ticketId")

	if ticketId == "" {
		logx.Logger.Info("ticketId parameter is missing in join request")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	reader, err := gameHandler.gameService.Join(gameId, ticketId, connection)

	if err != nil {
		// TODO: Add onJoinFailed handler to return decoded message
		err = connection.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
		if err != nil {
			logx.Logger.Error(
				err.Error(),
				zap.String("desc", "could not write error message to websocket"),
			)
		}

		err = connection.Close()
		if err != nil {
			logx.Logger.Error(
				err.Error(),
				zap.String("desc", "could not close websocket connection"),
			)
		}

		return
	}

	reader()
}
