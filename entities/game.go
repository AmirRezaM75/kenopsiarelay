package entities

import "github.com/amirrezam75/kenopsiarelay/pkg/syncx"

// GameState represents any game-specific state that can be stored in a game
type GameState interface{}

type Game[S GameState] struct {
	Id        string
	Status    string
	CreatorId string
	CreatedAt int64
	LobbyId   string
	State     S
	// I used map[] in order to easily remove player and load it in O(1)
	Players syncx.Map[string, *Player]
}

// TODO: OnPing and onLeft handlers
