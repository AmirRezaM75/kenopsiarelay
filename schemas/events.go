package schemas

import (
	"encoding/json"
)

type PublisherEvent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func GameCreatedEvent(gameId, lobbyId, gameSlug string) (string, error) {
	type GameCreatedContent struct {
		GameId   string `json:"gameId"`
		LobbyId  string `json:"lobbyId"`
		GameSlug string `json:"gameSlug"`
	}

	content := GameCreatedContent{
		GameId:   gameId,
		LobbyId:  lobbyId,
		GameSlug: gameSlug,
	}

	return encode("GameCreated", content)
}

func GameEndedEvent(gameId, lobbyId, gameSlug string) (string, error) {
	type GameEndedContent struct {
		GameId   string `json:"gameId"`
		LobbyId  string `json:"lobbyId"`
		GameSlug string `json:"gameSlug"`
	}

	content := GameEndedContent{
		GameId:   gameId,
		LobbyId:  lobbyId,
		GameSlug: gameSlug,
	}

	return encode("GameEnded", content)
}

func encode(eventType string, content any) (string, error) {
	message, err := json.Marshal(content)
	if err != nil {
		return "", err
	}

	event := PublisherEvent{
		Type:    eventType,
		Content: string(message),
	}

	e, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	return string(e), nil
}
