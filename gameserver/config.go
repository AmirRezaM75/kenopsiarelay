package gameserver

import (
	"context"

	"github.com/AmirRezaM75/kenopsiarelay/entities"
)

// Config contains all configuration options for the game server
type Config[S entities.GameState] struct {
	// LIFECYCLE MANAGEMENT: Context for controlling server shutdown
	// This context flows from user application down through all components
	// When cancelled, it triggers graceful shutdown of hub, players, and all goroutines
	Context context.Context

	// PERFORMANCE TUNING: Configure hub dispatch buffer size
	// Controls how many messages can be queued for dispatch before blocking
	// Higher values handle traffic spikes better but use more memory
	DispatchBufferSize int

	// GameSlug which is defined in GameData service
	GameSlug          string
	UserService       UserServiceConfig
	LobbyService      LobbyServiceConfig
	Publisher         PublisherConfig
	Router            RouterConfig
	OnMessageReceived entities.MessageReceivedHandler[S]
	OnPlayerJoined    entities.PlayerJoinedHandler[S]
	OnPlayerLeft      entities.PlayerLeftHandler[S]
	GameStateFactory  func() S
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
