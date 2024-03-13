package subscriber

import "github.com/0xnogo/messagerelayer/message"

type Subscriber struct {
	MsgType message.MessageType
	Ch      chan message.Message
}

func NewSubscriber(msgType message.MessageType, ch chan message.Message) Subscriber {
	return Subscriber{MsgType: msgType, Ch: ch}
}

func (s *Subscriber) IsInterestedIn(msgType message.MessageType) bool {
	return s.MsgType&msgType != 0
}
