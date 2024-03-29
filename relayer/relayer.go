package relayer

import (
	"log"
	"sync"

	"github.com/0xnogo/messagerelayer/buffer"
	"github.com/0xnogo/messagerelayer/message"
	"github.com/0xnogo/messagerelayer/socket"
	"github.com/0xnogo/messagerelayer/subscriber"
)

type MessageRelayer struct {
	socket          socket.NetworkSocket
	subscribers     []subscriber.Subscriber
	mux             sync.Mutex
	buffer          *buffer.RelayerBuffer
	messageReceived chan message.Message
	stopChannel     chan struct{}
	wg              sync.WaitGroup
	enableLogging   bool
}

func NewMessageRelayer(socket socket.NetworkSocket, enableLogging bool) *MessageRelayer {
	return &MessageRelayer{
		socket:          socket,
		messageReceived: make(chan message.Message),
		stopChannel:     make(chan struct{}),
		subscribers:     make([]subscriber.Subscriber, 0),
		buffer:          buffer.NewRelayBuffer(),
		enableLogging:   enableLogging,
	}
}

func (mr *MessageRelayer) SubscribeToMessages(msgType message.MessageType, ch chan message.Message) {
	mr.mux.Lock()
	defer mr.mux.Unlock()

	newSub := subscriber.NewSubscriber(msgType, ch)
	mr.subscribers = append(mr.subscribers, newSub)

	mr.catchUpSubscriber(newSub)
}

func (mr *MessageRelayer) Start() {
	if mr.enableLogging {
		log.Println("Starting the relayer...")
	}
	mr.wg.Add(2)

	go mr.readFromSocket()
	go mr.processMessage()
}

func (mr *MessageRelayer) Stop() {
	if mr.enableLogging {
		log.Println("Shutting down gracefully the relayer...")
	}
	mr.mux.Lock()
	defer mr.mux.Unlock()
	close(mr.stopChannel)

	// Wait for the read and process goroutines to finish
	mr.wg.Wait()

	close(mr.messageReceived)
	for _, sub := range mr.subscribers {
		close(sub.Ch)
	}
	if mr.enableLogging {
		log.Println("Relayer completely shut down.")
	}
}

// Read from the socket and send the message to the processing channel.
func (mr *MessageRelayer) readFromSocket() {
	defer mr.wg.Done()
	for {
		select {
		case <-mr.stopChannel:
			return
		default:
			msg, err := mr.socket.Read()
			if err != nil {
				// Potential improvement: max retry count
				if mr.enableLogging {
					log.Println("Error reading from socket:", err)
				}

				continue
			}

			// Use of select to achieve a graceful shutdown
			// If the messageReceived channel is full, it will block until it's empty
			// or until the relayer is stopped
			select {
			case mr.messageReceived <- msg:
			case <-mr.stopChannel:
				return
			}

		}
	}
}

// Process each incoming message and update the appropriate buffers.
func (mr *MessageRelayer) processMessage() {
	defer mr.wg.Done()
	for {
		select {
		case <-mr.stopChannel:
			return
		case msg := <-mr.messageReceived:
			// RelayBuffer is thread-safe
			mr.buffer.AddMessage(&msg)

			// Broadcast the message right away
			mr.broadcastMessage(&msg)
		}
	}
}

// Broadcast the message to all subscribers.
func (mr *MessageRelayer) broadcastMessage(msg *message.Message) {
	for _, sub := range mr.subscribers {
		if sub.IsInterestedIn(msg.Type) {
			mr.sendTo(sub, msg)
		}
	}
}

// Catch up a new subscriber with the messages already in the buffer.
func (mr *MessageRelayer) catchUpSubscriber(sub subscriber.Subscriber) {
	mr.buffer.IterateMessages(func(msg *message.Message) {
		if msg != nil && sub.IsInterestedIn(msg.Type) {
			mr.sendTo(sub, msg)
		}
	})
}

// Send the message to the subscriber.
func (mr *MessageRelayer) sendTo(sub subscriber.Subscriber, msg *message.Message) {
	select {
	case sub.Ch <- *msg:
		// Message sent
	default:
		// If the subscriber is busy, we don't want to block the relayer
		if mr.enableLogging {
			log.Println("Subscriber is busy")
		}
	}
}
