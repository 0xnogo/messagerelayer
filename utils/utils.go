package utils

import (
	"math/rand"

	"github.com/0xnogo/messagerelayer/message"
)

func FromIntToMsgTypeStr(msgType message.MessageType) string {
	switch msgType {
	case message.StartNewRound:
		return "StartNewRound"
	case message.ReceivedAnswer:
		return "ReceivedAnswer"
	case message.StartNewRound | message.ReceivedAnswer:
		return "Both"
	default:
		return "Unknown"
	}
}

func GenerateRandomMsgType(strict bool) message.MessageType {
	var n int
	if strict {
		n = 2
	} else {
		n = 3
	}

	switch rand.Intn(n) {
	case 0:
		return message.StartNewRound
	case 1:
		return message.ReceivedAnswer
	default:
		return message.StartNewRound | message.ReceivedAnswer
	}

}
