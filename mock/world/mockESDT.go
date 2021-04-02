package worldmock

import (
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/data/esdt"
)

// ESDTKeyPrefix is the prefix of storage keys belonging to ESDT tokens
var ESDTKeyPrefix = []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier)

// ESDTRoleKeyPrefix is the prefix of storage keys belonging to ESDT roles
var ESDTRoleKeyPrefix = []byte(core.ElrondProtectedKeyPrefix + core.ESDTRoleIdentifier + core.ESDTKeyIdentifier)

// GetTokenBalance returns the ESDT balance of an account for the given token
// key (token keys are built from the token identifier using MakeTokenKey).
func (bf *BuiltinFunctionsWrapper) GetTokenBalance(address []byte, tokenKey []byte) (*big.Int, error) {
	account := bf.World.AcctMap.GetAccount(address)
	return account.GetTokenBalance(tokenKey)
}

// SetTokenBalance sets the ESDT balance of an account for the given token
// key (token keys are built from the token identifier using MakeTokenKey).
func (bf *BuiltinFunctionsWrapper) SetTokenBalance(address []byte, tokenKey []byte, balance *big.Int) error {
	account := bf.World.AcctMap.GetAccount(address)
	return account.SetTokenBalance(tokenKey, balance)
}

// GetTokenData gets the ESDT information related to a token from the storage of an account
// (token keys are built from the token identifier using MakeTokenKey).
func (bf *BuiltinFunctionsWrapper) GetTokenData(address []byte, tokenKey []byte) (*esdt.ESDigitalToken, error) {
	account := bf.World.AcctMap.GetAccount(address)
	return account.GetTokenData(tokenKey)
}

// SetTokenData sets the ESDT information related to a token from the storage of an account
// (token keys are built from the token identifier using MakeTokenKey).
func (bf *BuiltinFunctionsWrapper) SetTokenData(address []byte, tokenKey []byte, tokenData *esdt.ESDigitalToken) error {
	account := bf.World.AcctMap.GetAccount(address)
	return account.SetTokenData(tokenKey, tokenData)
}

// PerformDirectESDTTransfer calls the real ESDTTransfer function immediately;
// only works for in-shard transfers for now, but it will be expanded to
// cross-shard.
// TODO rewrite to simulate what the SCProcessor does when executing a tx with
// data "ESDTTransfer@token@value@contractfunc@contractargs..."
func (bf *BuiltinFunctionsWrapper) PerformDirectESDTTransfer(
	sender []byte,
	receiver []byte,
	token []byte,
	value *big.Int,
	callType vmcommon.CallType,
	gasLimit uint64,
	gasPrice uint64,
) (uint64, error) {
	esdtTransferInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  sender,
			Arguments:   make([][]byte, 0),
			CallValue:   big.NewInt(0),
			CallType:    callType,
			GasPrice:    gasPrice,
			GasProvided: gasLimit,
			GasLocked:   0,
		},
		RecipientAddr:     receiver,
		Function:          core.BuiltInFunctionESDTTransfer,
		AllowInitFunction: false,
	}

	esdtTransferInput.Arguments = append(esdtTransferInput.Arguments, token, value.Bytes())
	vmOutput, err := bf.ProcessBuiltInFunction(esdtTransferInput)
	if err != nil {
		return 0, err
	}

	if vmOutput.ReturnCode != vmcommon.Ok {
		return 0, fmt.Errorf(
			"ESDTtransfer failed: retcode = %d, msg = %s",
			vmOutput.ReturnCode,
			vmOutput.ReturnMessage)
	}

	return vmOutput.GasRemaining, nil
}
