package relayer

import (
	"fmt"
	"sync"

	"github.com/0xnogo/messagerelayer/message"
	"github.com/0xnogo/messagerelayer/socket"
	"github.com/0xnogo/messagerelayer/subscriber"
)

type MessageRelayer struct {
	socket                socket.NetworkSocket
	subscribers           []subscriber.Subscriber
	mux                   sync.Mutex
	startNewRoundMessages [2]*message.Message // Buffer for the 2 most recent StartNewRound messages
	receivedAnswerMessage *message.Message    // Buffer for the most recent ReceivedAnswer message
	messageReceived       chan message.Message
	stopChannel           chan struct{}
}

func NewMessageRelayer(socket socket.NetworkSocket) *MessageRelayer {
	return &MessageRelayer{
		socket:          socket,
		messageReceived: make(chan message.Message),
		stopChannel:     make(chan struct{}),
		subscribers:     make([]subscriber.Subscriber, 0),
	}
}

func (mr *MessageRelayer) SubscribeToMessages(msgType message.MessageType, ch chan message.Message) {
	mr.mux.Lock()
	defer mr.mux.Unlock()

	newSub := subscriber.NewSubscriber(msgType, ch)
	mr.subscribers = append(mr.subscribers, newSub)

	mr.catchUpSubscriber(&newSub)
}

func (mr *MessageRelayer) Start() {
	go mr.readFromSocket()
	go mr.processMessage()
}

func (mr *MessageRelayer) Stop() {
	close(mr.stopChannel)
	close(mr.messageReceived)
	for _, sub := range mr.subscribers {
		close(sub.Ch)
	}
}

func (mr *MessageRelayer) readFromSocket() {
	for {
		select {
		case <-mr.stopChannel:
			return
		default:
			msg, err := mr.socket.Read()
			if err != nil {
				// handle error (log)
				continue
			}

			mr.messageReceived <- msg
		}
	}
}

// Process each incoming message and update the appropriate buffers.
func (mr *MessageRelayer) processMessage() {
	for {
		select {
		case <-mr.stopChannel:
			return
		case msg := <-mr.messageReceived:
			func() {
				mr.mux.Lock()
				defer mr.mux.Unlock()

				switch msg.Type {
				case message.StartNewRound:
					// Shift and insert to maintain only the 2 most recent messages
					mr.startNewRoundMessages[0] = mr.startNewRoundMessages[1]
					mr.startNewRoundMessages[1] = &msg
				case message.ReceivedAnswer:
					mr.receivedAnswerMessage = &msg
				}

				// Broadcast the message right away
				mr.broadcastMessage(&msg)
			}()
		}
	}
}

// Broadcast the message to all subscribers.
func (mr *MessageRelayer) broadcastMessage(msg *message.Message) {
	for _, sub := range mr.subscribers {
		if sub.IsInterestedIn(msg.Type) {
			mr.sendAndForget(&sub, msg)
		}
	}
}

// No need to lock here, as this is only called from within a locked block.
func (mr *MessageRelayer) catchUpSubscriber(sub *subscriber.Subscriber) {
	for _, msg := range mr.startNewRoundMessages {
		if msg != nil && sub.IsInterestedIn(msg.Type) {
			mr.sendAndForget(sub, msg)
		}
	}
	if mr.receivedAnswerMessage != nil && sub.IsInterestedIn(mr.receivedAnswerMessage.Type) {
		mr.sendAndForget(sub, mr.receivedAnswerMessage)
	}
}

func (mr *MessageRelayer) sendAndForget(sub *subscriber.Subscriber, msg *message.Message) {
	select {
	case sub.Ch <- *msg:
		// Message sent
	default:
		fmt.Println("Subscriber is busy")
		// If the subscriber is busy, we don't want to block the relayer
	}
}
