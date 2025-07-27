package gameserver

import "github.com/AmirRezaM75/kenopsiarelay/entities"

// Config contains all configuration options for the game server
type Config[S entities.GameState] struct {
	// GameSlug which is defined in GameData service
	GameSlug         string
	UserService      UserServiceConfig
	LobbyService     LobbyServiceConfig
	Publisher        PublisherConfig
	Router           RouterConfig
	OnMessage        entities.MessageHandler[S]
	GameStateFactory func() S
}

// UserServiceConfig contains configuration for the user service
type UserServiceConfig struct {
	BaseURL string
	Token   string
}

// LobbyServiceConfig contains configuration for the lobby service
type LobbyServiceConfig struct {
	BaseURL string
	Token   string
}

// PublisherConfig contains configuration for the publisher service
type PublisherConfig struct {
	Redis RedisConfig
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

// RouterConfig contains router configuration
type RouterConfig struct {
	AllowedOrigins []string
}
