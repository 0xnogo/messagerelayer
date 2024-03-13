package mocks

import (
	"fmt"
	"sync"
	"time"

	"github.com/0xnogo/messagerelayer/message"
	"github.com/0xnogo/messagerelayer/relayer"
)

// MockSubscriber simulates a subscriber for testing, with added message delay.
type MockSubscriber struct {
	Name               string
	MsgType            message.MessageType
	Ch                 chan message.Message
	StartNewRoundMsgs  []message.Message
	ReceivedAnswerMsgs []message.Message
	Delay              time.Duration // Processing delay to simulate business logic
	mux                sync.Mutex
}

// NewMockSubscriber creates a new instance of MockSubscriber.
func NewMockSubscriber(name string, msgType message.MessageType, delay time.Duration, bufferSize uint8) *MockSubscriber {
	return &MockSubscriber{
		Name:               name,
		MsgType:            msgType,
		Ch:                 make(chan message.Message, bufferSize), // Buffered channel
		StartNewRoundMsgs:  make([]message.Message, 0),
		ReceivedAnswerMsgs: make([]message.Message, 0),
		Delay:              delay,
	}
}

func (ms *MockSubscriber) Subscribe(relayer *relayer.MessageRelayer) *MockSubscriber {
	relayer.SubscribeToMessages(ms.MsgType, ms.Ch)
	return ms
}

// StartListening begins listening on the subscriber's channel for messages.
func (ms *MockSubscriber) StartListening() {
	go func() {
		for msg := range ms.Ch {
			fmt.Printf("Subscriber %s - Received message: %s\n", ms.Name, string(msg.Data))
			time.Sleep(ms.Delay) // Simulate processing delay
			ms.mux.Lock()
			if msg.Type == message.StartNewRound {
				ms.StartNewRoundMsgs = append(ms.StartNewRoundMsgs, msg)
			} else if msg.Type == message.ReceivedAnswer {
				ms.ReceivedAnswerMsgs = append(ms.ReceivedAnswerMsgs, msg)
			}
			ms.mux.Unlock()
		}
	}()
}

// IsInterestedIn checks if the subscriber is interested in a given message type.
func (ms *MockSubscriber) IsInterestedIn(msgType message.MessageType) bool {
	return ms.MsgType&msgType != 0
}
