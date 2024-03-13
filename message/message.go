package message

type MessageType int

const (
	StartNewRound MessageType = 1 << iota
	ReceivedAnswer
)

type Message struct {
	Type MessageType
	Data []byte
}

func NewMessage(msgType MessageType, data []byte) Message {
	return Message{Type: msgType, Data: data}
}
