package services

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/amirrezam75/kenopsialobby"
	"github.com/amirrezam75/kenopsiarelay/entities"
	"github.com/amirrezam75/kenopsiarelay/pkg/logx"
	"github.com/amirrezam75/kenopsiarelay/schemas"
	"github.com/amirrezam75/kenopsiauser"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

type GameService[S entities.GameState] struct {
	hub              *entities.Hub[S]
	userRepository   kenopsiauser.UserRepository
	lobbyRepository  kenopsialobby.LobbyRepository
	publisherService PublisherService
}

func NewGameService[S entities.GameState](
	hub *entities.Hub[S],
	userRepository kenopsiauser.UserRepository,
	lobbyRepository kenopsialobby.LobbyRepository,
	publisherService PublisherService,
) GameService[S] {
	return GameService[S]{
		hub:              hub,
		userRepository:   userRepository,
		lobbyRepository:  lobbyRepository,
		publisherService: publisherService,
	}
}

var (
	InvalidTicket  = errors.New("ticket is not valid")
	GameNotFound   = errors.New("game not found")
	PlayerNotFound = errors.New("player not found")
	LobbyNotFound  = errors.New("lobby not found")
)

func (gameService GameService[S]) Join(gameId, ticketId string, connection *websocket.Conn) (func(), error) {
	userId, err := gameService.userRepository.AcquireUserId(ticketId)

	if err != nil {
		logx.Logger.Error(
			err.Error(),
			zap.String("desc", "could not acquire user by ticket"),
		)
		return nil, InvalidTicket
	}

	game := gameService.hub.FindGame(gameId)

	if game == nil {
		return nil, GameNotFound
	}

	player, exists := game.Players.Load(userId)

	if !exists {
		return nil, PlayerNotFound
	}

	player.Kick()
	player.Message = make(chan []byte, 50)
	player.IsClosed = false
	player.Connection = connection
	player.IsConnected = true

	//gameService.hub.Dispatch <- schemas.SomeoneJoinedEvent(player.Id, game.Id)
	// TODO: onJoined()

	go player.Write()

	return func() {
		entities.Read(player, gameService.hub)
	}, nil
}

func (gameService GameService[S]) Create(
	user kenopsiauser.User,
	payload schemas.CreateGameRequest,
) (*schemas.CreateGameResponse, error) {
	lobby, err := gameService.lobbyRepository.FindById(payload.LobbyId)

	if err != nil {
		logx.Logger.Error(
			err.Error(),
			zap.String("lobbyId", payload.LobbyId),
			zap.String("desc", "could not find lobby by id"),
		)
		return nil, err
	}

	if lobby == nil {
		return nil, LobbyNotFound
	}

	game := &entities.Game[S]{
		Id:        bson.NewObjectID().Hex(),
		Status:    "pending",
		CreatorId: user.Id,
		CreatedAt: time.Now().Unix(),
		LobbyId:   lobby.Id,
		State:     gameService.hub.GameStateFactory(),
	}

	rand.Seed(time.Now().UnixNano())

	humansCount := len(lobby.Players)
	botsCount := len(lobby.Bots)

	indexes := rand.Perm(humansCount + botsCount)

	index := 0

	for _, player := range lobby.Players {
		game.Players.Store(player.Id, &entities.Player{
			Id:          player.Id,
			Username:    player.Username,
			GameId:      game.Id,
			AvatarId:    player.AvatarId,
			Index:       indexes[index] + 1,
			IsConnected: false,
			IsClosed:    true,
			IsBot:       false,
		})
		index++
	}

	for _, bot := range lobby.Bots {
		var botId = strconv.Itoa(int(bot.Id))

		game.Players.Store(botId, &entities.Player{
			Id:          botId,
			Username:    bot.Username,
			GameId:      game.Id,
			AvatarId:    bot.AvatarId,
			Index:       indexes[index] + 1,
			IsConnected: true,
			IsClosed:    true,
			IsBot:       true,
		})
		index++
	}

	gameService.hub.Games.Store(game.Id, game)

	message, err := schemas.GameCreatedEvent(game.Id, lobby.Id, "6nimmt") // TODO: Config

	if err != nil {
		logx.Logger.Error(
			err.Error(),
			zap.String("lobbyId", payload.LobbyId),
			zap.String("gameId", game.Id),
			zap.String("desc", "could not create GameCreatedEvent"),
		)
		return nil, err
	}

	err = gameService.publisherService.Publish(message)

	if err != nil {
		return nil, err
	}

	return &schemas.CreateGameResponse{GameId: game.Id}, nil
}
