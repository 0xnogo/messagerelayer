package subscriber

import (
	"testing"

	"github.com/0xnogo/messagerelayer/message"
)

func TestSubscriber_IsInterestedIn(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name       string
		subMsgType message.MessageType
		msgType    message.MessageType
		want       bool
	}{
		{
			name:       "Interested in StartNewRound",
			subMsgType: message.StartNewRound,
			msgType:    message.StartNewRound,
			want:       true,
		},
		{
			name:       "Not interested in ReceivedAnswer",
			subMsgType: message.StartNewRound,
			msgType:    message.ReceivedAnswer,
			want:       false,
		},
		{
			name:       "Interested in both",
			subMsgType: message.StartNewRound | message.ReceivedAnswer,
			msgType:    message.ReceivedAnswer,
			want:       true,
		},
		{
			name:       "Not interested in any",
			subMsgType: 0,
			msgType:    message.StartNewRound,
			want:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup subscriber with specified message type interest
			subscriber := NewSubscriber(tc.subMsgType, nil) // Channel is nil as it's not tested here

			// Test IsInterestedIn method
			got := subscriber.IsInterestedIn(tc.msgType)
			if got != tc.want {
				t.Errorf("IsInterestedIn(%v) = %v, want %v", tc.msgType, got, tc.want)
			}
		})
	}
}
