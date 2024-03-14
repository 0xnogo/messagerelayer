package mocks

import (
	"errors"
	"fmt"
	"time"

	"github.com/0xnogo/messagerelayer/message"
	"github.com/stretchr/testify/mock"
)

type MockNetworkSocket struct {
	mock.Mock
}

func (skt *MockNetworkSocket) Read() (message.Message, error) {
	args := skt.Called()

	return args.Get(0).(message.Message), args.Error(1)
}

// StaticMockNetworkSocket is a mock network socket that sends a fixed number of messages
type StaticMockNetworkSocket struct {
	totalMessages        int
	initialTotalMessages int
	timeout              time.Duration // Timeout after which EOF is returned if no new messages
}

func NewStaticMockNetworkSocket(totalMessages int, timeout time.Duration) *StaticMockNetworkSocket {
	return &StaticMockNetworkSocket{
		totalMessages:        totalMessages,
		initialTotalMessages: totalMessages,
		timeout:              timeout,
	}
}

func (m *StaticMockNetworkSocket) Read() (message.Message, error) {
	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms for a new message
	defer ticker.Stop()

	timeout := time.After(m.timeout)

	for {
		select {
		case <-ticker.C: // Check for new messages every tick
			if m.totalMessages > 0 {
				currentMessageIndex := m.initialTotalMessages - m.totalMessages

				msgType := message.StartNewRound
				if currentMessageIndex%2 != 0 {
					msgType = message.ReceivedAnswer
				}
				msg := message.NewMessage(msgType, []byte(fmt.Sprintf("Message %d", currentMessageIndex)))

				m.totalMessages--

				return msg, nil
			} else {
				// No more messages to send
				return message.Message{}, errors.New("EOF")
			}
		case <-timeout:
			return message.Message{}, errors.New("EOF")
		}
	}
}

// ChannelMockNetworkSocket is a mock network socket that sends messages from a channel
type ChannelMockNetworkSocket struct {
	messages chan message.Message
}

func NewChannelMockNetworkSocket() *ChannelMockNetworkSocket {
	return &ChannelMockNetworkSocket{
		messages: make(chan message.Message, 100),
	}
}

func (m *ChannelMockNetworkSocket) Read() (message.Message, error) {
	msg, ok := <-m.messages
	if !ok {
		return message.Message{}, errors.New("EOF")
	}
	return msg, nil
}

func (m *ChannelMockNetworkSocket) AddMessage(msg message.Message) {
	m.messages <- msg // Add a message to the channel
}

func (m *ChannelMockNetworkSocket) Close() {
	close(m.messages)
}
