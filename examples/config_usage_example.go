package examples

import (
	"encoding/json"
	"net/http"

	"github.com/amirrezam75/kenopsiarelay/entities"
	"github.com/amirrezam75/kenopsiarelay/pkg/gameserver"
)

// SixNimmtGameState represents the state for a 6 Nimmt game
type SixNimmtGameState struct {
	Round         int
	Trick         int
	Deck          []int
	Table         [][]int
	CurrentPlayer int
	GamePhase     string // "dealing", "playing", "scoring"
}

// NewSixNimmtGameState creates a new game state for 6 Nimmt
func NewSixNimmtGameState() SixNimmtGameState {
	return SixNimmtGameState{
		Round:         1,
		Trick:         1,
		Deck:          make([]int, 0),
		Table:         make([][]int, 4), // 4 rows on the table
		CurrentPlayer: 0,
		GamePhase:     "dealing",
	}
}

// Message handler for 6 Nimmt game - DIRECT ACCESS TO TYPED STATE!
func SixNimmtMessageHandler(hub *entities.Hub[SixNimmtGameState], game *entities.Game[SixNimmtGameState], player *entities.Player, message []byte) error {
	// Parse the message
	var request struct {
		Action string `json:"action"`
		Card   int    `json:"card,omitempty"`
	}

	if err := json.Unmarshal(message, &request); err != nil {
		return err
	}

	// DIRECT ACCESS TO TYPED STATE - NO TYPE ASSERTIONS!
	switch request.Action {
	case "play_card":
		// Direct access to typed state
		game.State.Round++
		game.State.GamePhase = "playing"

		// You can access any field directly:
		// game.State.Deck, game.State.Table, game.State.CurrentPlayer, etc.

	case "start_game":
		game.State.GamePhase = "playing"
		game.State.Deck = []int{1, 2, 3, 4, 5} // Example: initialize deck
	}

	return nil
}

// ExampleConfigUsage shows how to use the new config-based approach
func ExampleConfigUsage() {
	// 1. Create configuration with OnMessage and GameStateFactory
	config := gameserver.Config[SixNimmtGameState]{
		GameSlug: "sixnimmt",
		UserService: gameserver.UserServiceConfig{
			BaseURL: "https://kenopsia-user.example.com",
			Token:   "your-user-service-token",
		},
		LobbyService: gameserver.LobbyServiceConfig{
			BaseURL: "https://kenopsia-lobby.example.com",
			Token:   "your-lobby-service-token",
		},
		Publisher: gameserver.PublisherConfig{
			Redis: gameserver.RedisConfig{
				Host:     "localhost",
				Port:     "6379",
				Password: "",
			},
		},
		Router: gameserver.RouterConfig{
			AllowedOrigins: []string{"http://localhost:3000"},
		},
		// Game-specific handlers
		OnMessage:        SixNimmtMessageHandler,
		GameStateFactory: NewSixNimmtGameState,
	}

	// 2. Create game server in ONE LINE!
	gameServer := gameserver.NewGameServer(config)

	// 3. Get the router and start your HTTP server
	router := gameServer.GetRouter()

	// 4. Start your server
	http.ListenAndServe(":8080", router)
}

// ExampleConfigFromJSON shows how to load config from JSON
func ExampleConfigFromJSON() {
	configJSON := `{
		"gameSlug": "sixnimmt",
		"userService": {
			"baseUrl": "https://kenopsia-user.example.com",
			"token": "your-user-service-token"
		},
		"lobbyService": {
			"baseUrl": "https://kenopsia-lobby.example.com", 
			"token": "your-lobby-service-token"
		},
		"publisher": {
			"redis": {
				"host": "localhost",
				"port": "6379",
				"password": ""
			}
		},
		"router": {
			"allowedOrigins": ["http://localhost:3000"]
		}
	}`

	var config gameserver.Config[SixNimmtGameState]
	json.Unmarshal([]byte(configJSON), &config)

	// Add the handlers (cannot be serialized)
	config.OnMessage = SixNimmtMessageHandler
	config.GameStateFactory = NewSixNimmtGameState

	// Use the config
	gameServer := gameserver.NewGameServer(config)

	_ = gameServer // Use the game server
}

// This shows the COMPLETE solution:
// 1. Define your game state type (SixNimmtGameState)
// 2. Create a factory function (NewSixNimmtGameState)
// 3. Create a message handler with DIRECT ACCESS to game.State fields
// 4. Create a config struct with all your settings
// 5. Call gameserver.NewGameServer() - everything is set up!
// 6. Get the router with gameServer.GetRouter()
// 7. NO BOILERPLATE CODE IN YOUR MAIN!
