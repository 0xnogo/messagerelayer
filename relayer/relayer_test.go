package relayer_test

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/0xnogo/messagerelayer/message"
	"github.com/0xnogo/messagerelayer/relayer"
	mockSocket "github.com/0xnogo/messagerelayer/socket/mocks"
	mockSubcriber "github.com/0xnogo/messagerelayer/subscriber/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TODO:
// [] improve the waiting logic (wg?)
// [] scenario builder for the tests or factorize (sub creation, message creation) better
// [] Create a chaos test (add a test which send multiple messages with random times (blocker: waiting logic))

type RelayerTestSuite struct {
	suite.Suite
	mockSocket *mockSocket.MockNetworkSocket
	relayer    *relayer.MessageRelayer
}

func (rts *RelayerTestSuite) SetupTest() {
	rts.mockSocket = new(mockSocket.MockNetworkSocket)
	rts.relayer = relayer.NewMessageRelayer(rts.mockSocket)
}

func (rts *RelayerTestSuite) AfterTest() {
	rts.relayer.Stop()
}

func (rts *RelayerTestSuite) TestMessageRelayer_BroadcastsCorrectMessagesWithStartNewRound() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Round 1")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Round 2")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	// Create subscribers for the relayer.
	sub := mockSubcriber.NewMockSubscriber("1", message.StartNewRound, 0, 0)
	sub.StartListening()
	rts.relayer.SubscribeToMessages(sub.MsgType, sub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	time.Sleep(1 * time.Second)
	assert.Equal(rts.T(), 2, len(sub.StartNewRoundMsgs))
	assert.Equal(rts.T(), 0, len(sub.ReceivedAnswerMsgs))
}

func (rts *RelayerTestSuite) TestMessageRelayer_BroadcastsCorrectMessagesWithReceiveAnswer() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 1")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 2")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	// Create subscribers for the relayer.
	sub := mockSubcriber.NewMockSubscriber("1", message.ReceivedAnswer, 0, 0)
	sub.StartListening()
	rts.relayer.SubscribeToMessages(sub.MsgType, sub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	time.Sleep(1 * time.Second)
	assert.Equal(rts.T(), 2, len(sub.ReceivedAnswerMsgs))
	assert.Equal(rts.T(), 0, len(sub.StartNewRoundMsgs))
}

func (rts *RelayerTestSuite) TestMessageRelayer_BroadcastsMessageWithSubscriberInterestedIntoBothTypes() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 1")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Round 1")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	// Create subscribers for the relayer.
	sub := mockSubcriber.NewMockSubscriber("1", message.StartNewRound|message.ReceivedAnswer, 0, 0) // Interested in StartNewRound messages.
	sub.StartListening()
	rts.relayer.SubscribeToMessages(sub.MsgType, sub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	time.Sleep(1 * time.Second)
	assert.Equal(rts.T(), 1, len(sub.ReceivedAnswerMsgs))
	assert.Equal(rts.T(), 1, len(sub.StartNewRoundMsgs))
}

func (rts *RelayerTestSuite) TestMessageRelayer_BroadcastsMessageWithPreservedOrder() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 1")), nil).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 2")), nil).After(10 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Round 1")), nil).After(20 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Round 2")), nil).After(30 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	// Create subscribers for the relayer.
	sub := mockSubcriber.NewMockSubscriber("1", message.StartNewRound|message.ReceivedAnswer, 0, 0) // Interested in StartNewRound messages.
	sub.StartListening()

	rts.relayer.SubscribeToMessages(sub.MsgType, sub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	time.Sleep(1 * time.Second)
	assert.Equal(rts.T(), 2, len(sub.ReceivedAnswerMsgs))
	assert.Equal(rts.T(), 2, len(sub.StartNewRoundMsgs))

	// Verify the order of insertion
	for i, msg := range sub.ReceivedAnswerMsgs {
		assert.Equal(rts.T(), message.ReceivedAnswer, msg.Type)
		assert.Equal(rts.T(), []byte("Answer "+fmt.Sprint(i+1)), msg.Data)
	}

	for i, msg := range sub.StartNewRoundMsgs {
		assert.Equal(rts.T(), message.StartNewRound, msg.Type)
		assert.Equal(rts.T(), []byte("Round "+fmt.Sprint(i+1)), msg.Data)
	}
}

func (rts *RelayerTestSuite) TestMessageRelayer_BroadcastsCorrectMessagesWithRandomTypesAndMultipleSubscribers() {
	// GIVEN
	expectedResult := randomizeMessages(10, rts.mockSocket)

	// Create subscribers for the relayer
	sub1 := mockSubcriber.NewMockSubscriber("1", message.StartNewRound, 0, 0) // Interested in StartNewRound messages.
	sub1.StartListening()
	sub2 := mockSubcriber.NewMockSubscriber("1", message.ReceivedAnswer, 0, 0) // Interested in StartNewRound messages.
	sub2.StartListening()
	sub3 := mockSubcriber.NewMockSubscriber("1", message.StartNewRound|message.ReceivedAnswer, 0, 0) // Interested in StartNewRound messages.
	sub3.StartListening()
	subs := []*mockSubcriber.MockSubscriber{sub1, sub2, sub3}

	rts.relayer.SubscribeToMessages(sub1.MsgType, sub1.Ch)
	rts.relayer.SubscribeToMessages(sub2.MsgType, sub2.Ch)
	rts.relayer.SubscribeToMessages(sub3.MsgType, sub3.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	time.Sleep(1 * time.Second)
	for _, sub := range subs {
		if sub.IsInterestedIn(message.StartNewRound) {
			assert.Equal(rts.T(), len(expectedResult[message.StartNewRound]), len(sub.StartNewRoundMsgs))
		}
		if sub.IsInterestedIn(message.ReceivedAnswer) {
			assert.Equal(rts.T(), len(expectedResult[message.ReceivedAnswer]), len(sub.ReceivedAnswerMsgs))
		}
	}
}

func (rts *RelayerTestSuite) TestMessageRelayer_SubscribeToMessagesAfterStartSendsBufferedMessagesAndTheFollowingMessages() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 1")), nil).After(50 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 2")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 3")), nil).After(500 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	subBefore := mockSubcriber.NewMockSubscriber("Before", message.StartNewRound, 0, 0)
	subAfter := mockSubcriber.NewMockSubscriber("After", message.StartNewRound, 0, 2) // need a buffer of 2 to receive the first 2 messages
	subBefore.StartListening()
	subAfter.StartListening()

	// subscribed before the messages are sent
	rts.relayer.SubscribeToMessages(subBefore.MsgType, subBefore.Ch)

	// Subscribe after the relayer has started.
	time.AfterFunc(300*time.Millisecond, func() {
		rts.relayer.SubscribeToMessages(subAfter.MsgType, subAfter.Ch)
	})

	// WHEN
	rts.relayer.Start()

	// THEN
	time.Sleep(1 * time.Second)
	assert.Equal(rts.T(), 3, len(subBefore.StartNewRoundMsgs))
	assert.Equal(rts.T(), 3, len(subAfter.StartNewRoundMsgs))
}

func (rts *RelayerTestSuite) TestMessageRelayer_OnlyLastTwoStartNewRoundAndLastReceivedAnswerShouldBeBroadcastedToNewSub() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 1")), nil).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 1")), nil).After(50 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 2")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 3")), nil).After(150 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 2")), nil).After(200 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	subBefore := mockSubcriber.NewMockSubscriber("Before", message.StartNewRound|message.ReceivedAnswer, 0, 0)
	subAfter := mockSubcriber.NewMockSubscriber("After", message.StartNewRound|message.ReceivedAnswer, 0, 2) // need a buffer of 2 to receive the first 2 messages
	subBefore.StartListening()
	subAfter.StartListening()

	// subscribed before the messages are sent
	rts.relayer.SubscribeToMessages(subBefore.MsgType, subBefore.Ch)

	// delay the subscription to ensure that the buffered messages are sent
	time.AfterFunc(700*time.Millisecond, func() {
		rts.relayer.SubscribeToMessages(subAfter.MsgType, subAfter.Ch)
	})

	// WHEN
	rts.relayer.Start()

	// THEN
	time.Sleep(1 * time.Second)

	assert.Equal(rts.T(), 3, len(subBefore.StartNewRoundMsgs))
	assert.Equal(rts.T(), 2, len(subAfter.StartNewRoundMsgs)) // instead of 3

	assert.Equal(rts.T(), 2, len(subBefore.ReceivedAnswerMsgs))
	assert.Equal(rts.T(), 1, len(subAfter.ReceivedAnswerMsgs)) // instead of 2

	assert.Equal(rts.T(), "Start 2", string(subAfter.StartNewRoundMsgs[0].Data))
	assert.Equal(rts.T(), "Start 3", string(subAfter.StartNewRoundMsgs[1].Data))

	assert.Equal(rts.T(), "Answer 2", string(subAfter.ReceivedAnswerMsgs[0].Data))
}

func (rts *RelayerTestSuite) TestMessageRelayer_BroadcastsMessagesWithBusySubscriber() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 1")), nil).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 2")), nil).After(50 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 3")), nil).After(200 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	slowSub := mockSubcriber.NewMockSubscriber("Slow", message.StartNewRound, 100*time.Millisecond, 0)
	fastSub := mockSubcriber.NewMockSubscriber("Fast", message.StartNewRound, 0, 0)

	slowSub.StartListening()
	fastSub.StartListening()

	rts.relayer.SubscribeToMessages(slowSub.MsgType, slowSub.Ch)
	rts.relayer.SubscribeToMessages(fastSub.MsgType, fastSub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	time.Sleep(1 * time.Second)
	assert.Equal(rts.T(), 3, len(fastSub.StartNewRoundMsgs))
	assert.Equal(rts.T(), 2, len(slowSub.StartNewRoundMsgs)) // instead of 3 - "Start 2" is not received
}

func TestRelayerTestSuite(t *testing.T) {
	suite.Run(t, new(RelayerTestSuite))
}

func randomizeMessages(numberOfMessages int, mockSocket *mockSocket.MockNetworkSocket) map[message.MessageType][]string {
	var expectedResult map[message.MessageType][]string = make(map[message.MessageType][]string)

	for i := 1; i < numberOfMessages; i++ {
		var msg message.Message
		if rand.Intn(2) == 0 { // Randomly choose between 0 and 1 to decide the message type.
			msg = message.NewMessage(message.StartNewRound, []byte("Round "+fmt.Sprint(i)))
			expectedResult[message.StartNewRound] = append(expectedResult[message.StartNewRound], "Round "+fmt.Sprint(i))
		} else {
			msg = message.NewMessage(message.ReceivedAnswer, []byte("Answer "+fmt.Sprint(i)))
			expectedResult[message.ReceivedAnswer] = append(expectedResult[message.ReceivedAnswer], "Answer "+fmt.Sprint(i))
		}
		mockSocket.On("Read").Return(msg, nil).After(100 * time.Millisecond).Once()
	}

	mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	return expectedResult
}
