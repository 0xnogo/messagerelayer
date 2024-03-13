package mocks

import (
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
