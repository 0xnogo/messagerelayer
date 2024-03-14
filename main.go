package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/0xnogo/messagerelayer/message"
	"github.com/0xnogo/messagerelayer/relayer"
	"github.com/0xnogo/messagerelayer/socket/mocks"
	"github.com/0xnogo/messagerelayer/subscriber"
	"github.com/0xnogo/messagerelayer/utils"
	"github.com/nsf/termbox-go"
)

func main() {
	interactiveMode := flag.Bool("interactive", false, "Run in interactive mode with termbox")
	flag.Parse()

	if *interactiveMode {
		runInteractiveMode()
	} else {
		runNormalMode()
	}
}

func runNormalMode() {
	totalMessages := 30
	totalSubscribers := 3

	mockSocket := mocks.NewStaticMockNetworkSocket(totalMessages, 2*time.Second)
	rel := relayer.NewMessageRelayer(mockSocket, true)

	// Subscribe users with random interests
	for i := 0; i < totalSubscribers; i++ {
		ch := make(chan message.Message)
		var msgType message.MessageType = message.MessageType(i + 1)

		rel.SubscribeToMessages(msgType, ch)

		// print the user's interest
		log.Printf("Subscriber %d is interested in: %s\n", i+1, utils.FromIntToMsgTypeStr(msgType))
		go func(subMsgType message.MessageType, subID int, ch chan message.Message) {
			for msg := range ch {
				log.Printf("Subscriber %d (interested in %s) received: %s - type: %s \n", subID, utils.FromIntToMsgTypeStr(subMsgType), string(msg.Data), utils.FromIntToMsgTypeStr(msg.Type))
			}
		}(msgType, i+1, ch)
	}

	go func() {
		time.Sleep(5 * time.Second) // Wait for a few seconds before adding a subscriber

		// Simulate adding a new subscriber
		ch := make(chan message.Message, 10)
		var msgType message.MessageType = utils.GenerateRandomMsgType(true)

		rel.SubscribeToMessages(msgType, ch)

		// print the user's interest
		log.Printf("===============")
		log.Printf("New subscriber is interested in: %s\n", utils.FromIntToMsgTypeStr(msgType))
		go func(subMsgType message.MessageType, ch chan message.Message) {
			for msg := range ch {
				log.Printf("New subscriber (interested in %s) received: %s - type: %s \n", utils.FromIntToMsgTypeStr(subMsgType), string(msg.Data), utils.FromIntToMsgTypeStr(msg.Type))
			}
		}(msgType, ch)
	}()

	// Start processing messages
	rel.Start()

	// Wait for a few seconds before shutting down
	time.Sleep(6 * time.Second)

	// Shut down the relayer
	rel.Stop()
}

func runInteractiveMode() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	subscribers := make([]subscriber.Subscriber, 0)
	messageCounter := 0

	fmt.Println("Starting message relayer simulation...")
	fmt.Println("Press 'm' to add a message, 'u' to add a user, 'l' to print the list of subscribers and 'Ctrl+c' to quit.")

	mockSocket := mocks.NewChannelMockNetworkSocket()
	rel := relayer.NewMessageRelayer(mockSocket, true)
	rel.Start()

loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyCtrlC, termbox.KeyEsc:
				break loop // Exit loop on Ctrl+C or Esc
			default:
				handleKeyPress(ev.Ch, rel, mockSocket, &subscribers, &messageCounter)
			}
		}
	}

	// Clean up
	mockSocket.Close()
	rel.Stop()
}

func handleKeyPress(ch rune, rel *relayer.MessageRelayer, socket *mocks.ChannelMockNetworkSocket, subscribers *[]subscriber.Subscriber, messageCounter *int) {
	switch ch {
	case 'm':
		msgType := utils.GenerateRandomMsgType(true)
		msgContent := []byte(fmt.Sprintf("Message %d", *messageCounter))
		newMsg := message.NewMessage(msgType, msgContent)
		socket.AddMessage(newMsg)
		fmt.Printf("Added new message of type %s with content: %s\n", utils.FromIntToMsgTypeStr(msgType), string(msgContent))
		*messageCounter++

	case 'u':
		ch := make(chan message.Message, 10)
		msgType := utils.GenerateRandomMsgType(false)
		*subscribers = append(*subscribers, subscriber.NewSubscriber(msgType, ch))
		rel.SubscribeToMessages(msgType, ch)

		fmt.Printf("Added new subscriber: interested in %s\n", utils.FromIntToMsgTypeStr(msgType))

		// Start a new goroutine to listen for messages on the new subscriber's channel
		go func(sub *subscriber.Subscriber) {
			for msg := range ch {
				fmt.Printf("Subscriber (%s) received message (Type: %s, Content: %s)\n", utils.FromIntToMsgTypeStr(sub.MsgType), utils.FromIntToMsgTypeStr(msg.Type), string(msg.Data))
			}
		}(&(*subscribers)[len(*subscribers)-1])

	case 'l':
		// List all subscribers
		if len(*subscribers) == 0 {
			fmt.Println("No subscribers. Click 'u' to add a new subscriber.")
		} else {
			fmt.Println("Listing all subscribers:")
			for i, sub := range *subscribers {
				fmt.Printf("Subscriber %d - interested into %s\n", i, utils.FromIntToMsgTypeStr(sub.MsgType))
			}
		}

	default:
		fmt.Println("Unknown command.")
	}
}
