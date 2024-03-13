package socket

import (
	"github.com/0xnogo/messagerelayer/message"
)

type NetworkSocket interface {
	Read() (message.Message, error)
}
