package handlers

import (
	"errors"
	"net/http"
	"os"

	middlewares "github.com/amirrezam75/kenopsiacommon/middlwares"
	commonservices "github.com/amirrezam75/kenopsiacommon/services"
	"github.com/amirrezam75/kenopsiarelay/pkg/logx"
	"github.com/amirrezam75/kenopsiarelay/schemas"
	"github.com/amirrezam75/kenopsiarelay/services"
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
		logx.Logger.Info(err, zap.String("desc", "could not decode payload"))
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
		logx.Logger.Info("ticketId is not provided")
		w.WriteHeader(422)
		return
	}

	reader, err := gameHandler.gameService.Join(gameId, ticketId, connection)

	if err != nil {
		/*message := schemas.ReportErrorMessage(err.Error())

		binary, err := proto.Marshal(&message)*/
		// TODO: onJoinFailure
		if err != nil {
			logx.Logger.Error(
				err.Error(),
				zap.String("desc", "could not marshal message"),
			)
			return
		}

		err = connection.WriteMessage(websocket.BinaryMessage, []byte("")) // TODO: use binary variable

		if err != nil {
			logx.Logger.Error(
				err.Error(),
				zap.String("desc", "could not write message"),
			)
		}
	}

	reader()
}
