package relayer_test

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/0xnogo/messagerelayer/message"
	"github.com/0xnogo/messagerelayer/relayer"
	mockSocket "github.com/0xnogo/messagerelayer/socket/mocks"
	mockSubcriber "github.com/0xnogo/messagerelayer/subscriber/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RelayerTestSuite struct {
	suite.Suite
	mockSocket *mockSocket.MockNetworkSocket
	relayer    *relayer.MessageRelayer
	skipStop   bool
}

func (rts *RelayerTestSuite) SetupTest() {
	rts.mockSocket = new(mockSocket.MockNetworkSocket)
	rts.relayer = relayer.NewMessageRelayer(rts.mockSocket, false)
	rts.skipStop = false
}

func (rts *RelayerTestSuite) TearDownTest() {
	if !rts.skipStop {
		rts.relayer.Stop()
	}
}

func (rts *RelayerTestSuite) TestBroadcastStartNewRound() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Round 1")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Round 2")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	var wg sync.WaitGroup
	wg.Add(2) // total number of messages expected

	// Create subscribers for the relayer.
	sub := mockSubcriber.NewMockSubscriber("1", message.StartNewRound, 0, 0, &wg)
	sub.StartListening()
	rts.relayer.SubscribeToMessages(sub.MsgType, sub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	wg.Wait()
	assert.Equal(rts.T(), 2, len(sub.StartNewRoundMsgs))
	assert.Equal(rts.T(), 0, len(sub.ReceivedAnswerMsgs))
}

func (rts *RelayerTestSuite) TestBroadcastReceivedAnswer() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 1")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 2")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	var wg sync.WaitGroup
	wg.Add(2) // total number of messages expected

	// Create subscribers for the relayer.
	sub := mockSubcriber.NewMockSubscriber("1", message.ReceivedAnswer, 0, 0, &wg)
	sub.StartListening()
	rts.relayer.SubscribeToMessages(sub.MsgType, sub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	wg.Wait()
	assert.Equal(rts.T(), 2, len(sub.ReceivedAnswerMsgs))
	assert.Equal(rts.T(), 0, len(sub.StartNewRoundMsgs))
}

func (rts *RelayerTestSuite) TestBroadcastToMultiInterestSubscriber() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.ReceivedAnswer, []byte("Answer 1")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Round 1")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	var wg sync.WaitGroup
	wg.Add(2) // total number of messages expected

	// Create subscribers for the relayer.
	sub := mockSubcriber.NewMockSubscriber("1", message.StartNewRound|message.ReceivedAnswer, 0, 0, &wg)
	sub.StartListening()
	rts.relayer.SubscribeToMessages(sub.MsgType, sub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	wg.Wait()
	assert.Equal(rts.T(), 1, len(sub.ReceivedAnswerMsgs))
	assert.Equal(rts.T(), 1, len(sub.StartNewRoundMsgs))
}

func (rts *RelayerTestSuite) TestPreserveMessageOrder() {
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

	var wg sync.WaitGroup
	wg.Add(4) // total number of messages expected

	// Create subscribers for the relayer.
	sub := mockSubcriber.NewMockSubscriber("1", message.StartNewRound|message.ReceivedAnswer, 0, 0, &wg)
	sub.StartListening()

	rts.relayer.SubscribeToMessages(sub.MsgType, sub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	wg.Wait()
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

func (rts *RelayerTestSuite) TestBroadcastRandomToMultipleSubs() {
	// GIVEN
	expectedResult := randomizeMessages(10, rts.mockSocket)

	var wg sync.WaitGroup
	rts.T().Log(len(expectedResult[message.StartNewRound])*2 + len(expectedResult[message.ReceivedAnswer])*2)
	wg.Add(len(expectedResult[message.StartNewRound])*2 + len(expectedResult[message.ReceivedAnswer])*2) // total number of messages expected

	// Create subscribers for the relayer
	sub1 := mockSubcriber.NewMockSubscriber("1", message.StartNewRound, 0, 0, &wg)
	sub1.StartListening()
	sub2 := mockSubcriber.NewMockSubscriber("1", message.ReceivedAnswer, 0, 0, &wg)
	sub2.StartListening()
	sub3 := mockSubcriber.NewMockSubscriber("1", message.StartNewRound|message.ReceivedAnswer, 0, 0, &wg)
	sub3.StartListening()
	subs := []*mockSubcriber.MockSubscriber{sub1, sub2, sub3}

	rts.relayer.SubscribeToMessages(sub1.MsgType, sub1.Ch)
	rts.relayer.SubscribeToMessages(sub2.MsgType, sub2.Ch)
	rts.relayer.SubscribeToMessages(sub3.MsgType, sub3.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	wg.Wait()
	for _, sub := range subs {
		if sub.IsInterestedIn(message.StartNewRound) {
			assert.Equal(rts.T(), len(expectedResult[message.StartNewRound]), len(sub.StartNewRoundMsgs))
		}
		if sub.IsInterestedIn(message.ReceivedAnswer) {
			assert.Equal(rts.T(), len(expectedResult[message.ReceivedAnswer]), len(sub.ReceivedAnswerMsgs))
		}
	}
}

func (rts *RelayerTestSuite) TestLateSubscribeGetsBufferedAndNew() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 1")), nil).After(50 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 2")), nil).After(100 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 3")), nil).After(500 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	var wg sync.WaitGroup
	wg.Add(6) // total number of messages expected

	subBefore := mockSubcriber.NewMockSubscriber("Before", message.StartNewRound, 0, 0, &wg)
	subAfter := mockSubcriber.NewMockSubscriber("After", message.StartNewRound, 0, 2, &wg) // need a buffer of 2 to receive the first 2 messages
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
	wg.Wait()
	assert.Equal(rts.T(), 3, len(subBefore.StartNewRoundMsgs))
	assert.Equal(rts.T(), 3, len(subAfter.StartNewRoundMsgs))
}

func (rts *RelayerTestSuite) TestNewSubGetsRecentMessages() {
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

	var wg sync.WaitGroup
	wg.Add(8) // total number of messages expected

	subBefore := mockSubcriber.NewMockSubscriber("Before", message.StartNewRound|message.ReceivedAnswer, 0, 0, &wg)
	subAfter := mockSubcriber.NewMockSubscriber("After", message.StartNewRound|message.ReceivedAnswer, 0, 2, &wg) // need a buffer of 2 to receive the first 2 messages
	subBefore.StartListening()
	subAfter.StartListening()

	// subscribed before the messages are sent
	rts.relayer.SubscribeToMessages(subBefore.MsgType, subBefore.Ch)

	// delay the subscription to ensure that the buffered messages are sent
	time.AfterFunc(800*time.Millisecond, func() {
		rts.relayer.SubscribeToMessages(subAfter.MsgType, subAfter.Ch)
	})

	// WHEN
	rts.relayer.Start()

	// THEN
	wg.Wait()

	assert.Equal(rts.T(), 3, len(subBefore.StartNewRoundMsgs))
	assert.Equal(rts.T(), 2, len(subAfter.StartNewRoundMsgs)) // instead of 3

	assert.Equal(rts.T(), 2, len(subBefore.ReceivedAnswerMsgs))
	assert.Equal(rts.T(), 1, len(subAfter.ReceivedAnswerMsgs)) // instead of 2

	assert.Equal(rts.T(), "Start 2", string(subAfter.StartNewRoundMsgs[0].Data))
	assert.Equal(rts.T(), "Start 3", string(subAfter.StartNewRoundMsgs[1].Data))

	assert.Equal(rts.T(), "Answer 2", string(subAfter.ReceivedAnswerMsgs[0].Data))
}

func (rts *RelayerTestSuite) TestBroadcastWithBusySubscriber() {
	// GIVEN
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 1")), nil).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 2")), nil).After(50 * time.Millisecond).Once()
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start 3")), nil).After(200 * time.Millisecond).Once()
	rts.mockSocket.On("Read").Return(message.Message{}, errors.New("EOF"))

	var wg sync.WaitGroup
	wg.Add(5) // total number of messages expected

	slowSub := mockSubcriber.NewMockSubscriber("Slow", message.StartNewRound, 100*time.Millisecond, 0, &wg)
	fastSub := mockSubcriber.NewMockSubscriber("Fast", message.StartNewRound, 0, 0, &wg)

	slowSub.StartListening()
	fastSub.StartListening()

	rts.relayer.SubscribeToMessages(slowSub.MsgType, slowSub.Ch)
	rts.relayer.SubscribeToMessages(fastSub.MsgType, fastSub.Ch)

	// WHEN
	rts.relayer.Start()

	// THEN
	wg.Wait()
	assert.Equal(rts.T(), 3, len(fastSub.StartNewRoundMsgs))
	assert.Equal(rts.T(), 2, len(slowSub.StartNewRoundMsgs)) // instead of 3 - "Start 2" is not received
}

func (rts *RelayerTestSuite) TestStopShutdownWithoutPanicWhileMessageAreSent() {
	rts.skipStop = true

	// GIVEN
	ch := make(chan message.Message, 100000)
	rts.relayer.SubscribeToMessages(message.StartNewRound, ch)
	rts.mockSocket.On("Read").
		Return(message.NewMessage(message.StartNewRound, []byte("Start")), nil)

	rts.relayer.Start()

	go func() {
		for {
			<-ch
		}
	}()

	// Allow some operations to potentially start or messages to be queued
	time.Sleep(100 * time.Millisecond)

	// WHEN
	rts.relayer.Stop()

	// THEN
	_, ok := <-ch
	require.False(rts.T(), ok, "Subscriber channel should be closed")
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
