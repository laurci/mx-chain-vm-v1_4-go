package nodepart

import (
	"os"
	"time"

	"github.com/ElrondNetwork/arwen-wasm-vm/ipc/common"
	"github.com/ElrondNetwork/arwen-wasm-vm/ipc/logger"
)

// NodeMessenger is the messenger on Node's part of the pipe
type NodeMessenger struct {
	common.Messenger
}

// NewNodeMessenger creates a new messenger
func NewNodeMessenger(logger logger.Logger, reader *os.File, writer *os.File) *NodeMessenger {
	return &NodeMessenger{
		Messenger: *common.NewMessenger("NODE", logger, reader, writer),
	}
}

// SendContractRequest sends a request to Arwen
func (messenger *NodeMessenger) SendContractRequest(request common.MessageHandler) error {
	err := messenger.Send(request)
	if err != nil {
		return common.ErrCannotSendContractRequest
	}

	return nil
}

// SendHookCallResponse sends a hook response to Arwen
func (messenger *NodeMessenger) SendHookCallResponse(response common.MessageHandler) error {
	err := messenger.Send(response)
	if err != nil {
		return common.ErrCannotSendHookCallResponse
	}

	return nil
}

// ReceiveHookCallRequestOrContractResponse waits for any message that could arrive from Arwen
func (messenger *NodeMessenger) ReceiveHookCallRequestOrContractResponse(timeout int) (common.MessageHandler, int, error) {
	start := time.Now()
	message, err := messenger.Receive(timeout)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return nil, int(duration), err
	}

	return message, int(duration), nil
}
