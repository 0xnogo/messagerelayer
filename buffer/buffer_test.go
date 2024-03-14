package buffer_test

import (
	"testing"

	"github.com/0xnogo/messagerelayer/buffer"
	"github.com/0xnogo/messagerelayer/message"
	"github.com/stretchr/testify/assert"
)

func TestAddMessage(t *testing.T) {
	// GIVEN
	relayBuf := buffer.NewRelayBuffer()

	msg1 := &message.Message{Type: message.StartNewRound, Data: []byte("Round 1")}
	msg2 := &message.Message{Type: message.StartNewRound, Data: []byte("Round 2")}
	msg3 := &message.Message{Type: message.ReceivedAnswer, Data: []byte("Answer")}

	// WHEN
	relayBuf.AddMessage(msg1)
	relayBuf.AddMessage(msg2)
	relayBuf.AddMessage(msg3)

	// THEN
	assert.Equal(t, relayBuf.StartNewRoundMessages()[0], msg1)
	assert.Equal(t, relayBuf.StartNewRoundMessages()[1], msg2)
	assert.Equal(t, relayBuf.ReceivedAnswerMessage(), msg3)
}

func TestEvictionPolicy(t *testing.T) {
	// GIVEN
	relayBuf := buffer.NewRelayBuffer()

	msg1 := &message.Message{Type: message.StartNewRound, Data: []byte("Round 1")}
	msg2 := &message.Message{Type: message.StartNewRound, Data: []byte("Round 2")}
	msg3 := &message.Message{Type: message.StartNewRound, Data: []byte("Round 3")}
	msg4 := &message.Message{Type: message.ReceivedAnswer, Data: []byte("Answer 1")}
	msg5 := &message.Message{Type: message.ReceivedAnswer, Data: []byte("Answer 2")}

	// WHEN
	relayBuf.AddMessage(msg1)
	relayBuf.AddMessage(msg2)
	relayBuf.AddMessage(msg3)
	relayBuf.AddMessage(msg4)
	relayBuf.AddMessage(msg5)

	// THEN
	assert.Equal(t, relayBuf.StartNewRoundMessages()[0], msg2)
	assert.Equal(t, relayBuf.StartNewRoundMessages()[1], msg3)
	assert.Equal(t, relayBuf.ReceivedAnswerMessage(), msg5)
}

func TestIterateMessages(t *testing.T) {
	// GIVEN
	relayBuf := buffer.NewRelayBuffer()

	msg1 := &message.Message{Type: message.StartNewRound, Data: []byte("Round 1")}
	msg2 := &message.Message{Type: message.StartNewRound, Data: []byte("Round 2")}
	msg3 := &message.Message{Type: message.ReceivedAnswer, Data: []byte("Answer 1")}
	msg4 := &message.Message{Type: message.ReceivedAnswer, Data: []byte("Answer 2")}
	msg5 := &message.Message{Type: message.StartNewRound, Data: []byte("Round 3")}

	relayBuf.AddMessage(msg1)
	relayBuf.AddMessage(msg2)
	relayBuf.AddMessage(msg3)
	relayBuf.AddMessage(msg4)
	relayBuf.AddMessage(msg5)

	// WHEN
	collectedMessages := make([]*message.Message, 0)
	relayBuf.IterateMessages(func(msg *message.Message) {
		collectedMessages = append(collectedMessages, msg)
	})

	// THEN
	assert.Len(t, collectedMessages, 3)
	assert.Equal(t, collectedMessages[0], msg2)
	assert.Equal(t, collectedMessages[1], msg5)
	assert.Equal(t, collectedMessages[2], msg4)
}
