package schemas

type DispatcherMessage struct {
	Body        []byte
	GameId      string
	ReceiverIds []string
}
