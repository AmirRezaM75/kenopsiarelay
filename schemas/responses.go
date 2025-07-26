package schemas

type CreateGameResponse struct {
	GameId string `json:"gameId"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
