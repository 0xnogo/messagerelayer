package buffer

import (
	"sync"

	"github.com/0xnogo/messagerelayer/message"
)

// Note for the reviewer:
// For performance reasons, it is implemented as a fixed-size buffer.
// Potential improvement could be to parameterize the buffer size
// but will involve performance trade-offs.

// RelayBuffer manages the storage and prioritization of messages.
type RelayBuffer struct {
	startNewRoundMessages [2]*message.Message // Buffer for the 2 most recent StartNewRound messages
	receivedAnswerMessage *message.Message    // Buffer for the most recent ReceivedAnswer message
	mux                   sync.RWMutex
}

func NewRelayBuffer() *RelayBuffer {
	return &RelayBuffer{}
}

func (rb *RelayBuffer) StartNewRoundMessages() [2]*message.Message {
	return rb.startNewRoundMessages
}

func (rb *RelayBuffer) ReceivedAnswerMessage() *message.Message {
	return rb.receivedAnswerMessage
}

// AddMessage adds a message to the buffer, following the eviction policy.
func (rb *RelayBuffer) AddMessage(msg *message.Message) {
	rb.mux.Lock()
	defer rb.mux.Unlock()

	switch msg.Type {
	case message.StartNewRound:
		rb.startNewRoundMessages[0] = rb.startNewRoundMessages[1]
		rb.startNewRoundMessages[1] = msg
	case message.ReceivedAnswer:
		rb.receivedAnswerMessage = msg
	}
}

// IterateMessages calls the given callback function for each message in the buffer,
// in the correct priority order.
func (rb *RelayBuffer) IterateMessages(callback func(*message.Message)) {
	rb.mux.RLock()
	defer rb.mux.RUnlock()

	for _, msg := range rb.startNewRoundMessages {
		if msg != nil {
			callback(msg)
		}
	}
	if rb.receivedAnswerMessage != nil {
		callback(rb.receivedAnswerMessage)
	}
}
