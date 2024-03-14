package utils_test

import (
	"testing"

	"github.com/0xnogo/messagerelayer/message"
	"github.com/0xnogo/messagerelayer/utils"
)

func TestFromIntToMsgTypeStr(t *testing.T) {
	tests := []struct {
		msgType message.MessageType
		want    string
	}{
		{message.StartNewRound, "StartNewRound"},
		{message.ReceivedAnswer, "ReceivedAnswer"},
		{message.StartNewRound | message.ReceivedAnswer, "Both"},
		{message.MessageType(999), "Unknown"}, // Test an undefined MessageType
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := utils.FromIntToMsgTypeStr(tt.msgType); got != tt.want {
				t.Errorf("FromIntToMsgTypeStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateRandomMsgType(t *testing.T) {
	const iterations = 1000
	foundTypes := make(map[message.MessageType]bool)

	for i := 0; i < iterations; i++ {
		got := utils.GenerateRandomMsgType(true)
		if got != message.StartNewRound && got != message.ReceivedAnswer {
			t.Errorf("GenerateRandomMsgType(true) returned unexpected value: %v", got)
		}
		foundTypes[got] = true
	}

	// Check that both possible values were generated
	if !foundTypes[message.StartNewRound] || !foundTypes[message.ReceivedAnswer] {
		t.Errorf("GenerateRandomMsgType(true) did not generate all expected values")
	}

	// Reset for non-strict test
	foundTypes = make(map[message.MessageType]bool)
	for i := 0; i < iterations; i++ {
		got := utils.GenerateRandomMsgType(false)
		if got != message.StartNewRound && got != message.ReceivedAnswer && got != (message.StartNewRound|message.ReceivedAnswer) {
			t.Errorf("GenerateRandomMsgType(false) returned unexpected value: %v", got)
		}
		foundTypes[got] = true
	}

	// Check that all three possible values were generated
	if !foundTypes[message.StartNewRound] || !foundTypes[message.ReceivedAnswer] || !foundTypes[message.StartNewRound|message.ReceivedAnswer] {
		t.Errorf("GenerateRandomMsgType(false) did not generate all expected values")
	}
}
