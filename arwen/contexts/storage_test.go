package contexts

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/wasm-vm-v1_4/arwen"
	"github.com/ElrondNetwork/wasm-vm-v1_4/arwen/mock"
	"github.com/ElrondNetwork/wasm-vm-v1_4/config"
	contextmock "github.com/ElrondNetwork/wasm-vm-v1_4/mock/context"
	worldmock "github.com/ElrondNetwork/wasm-vm-v1_4/mock/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var elrondReservedTestPrefix = []byte("RESERVED")

func TestNewStorageContext(t *testing.T) {
	t.Parallel()

	t.Run("empty protected key prefix should error", func(t *testing.T) {
		t.Parallel()

		host := &contextmock.VMHostMock{}
		mockBlockchain := worldmock.NewMockWorld()

		storageContextInstance, err := NewStorageContext(host, mockBlockchain, make([]byte, 0))
		require.Equal(t, arwen.ErrEmptyProtectedKeyPrefix, err)
		require.True(t, check.IfNil(storageContextInstance))
	})
	t.Run("nil VM host should error", func(t *testing.T) {
		t.Parallel()

		mockBlockchain := worldmock.NewMockWorld()

		storageContextInstance, err := NewStorageContext(nil, mockBlockchain, elrondReservedTestPrefix)
		require.Equal(t, arwen.ErrNilVMHost, err)
		require.True(t, check.IfNil(storageContextInstance))
	})
	t.Run("nil blockchain hook should error", func(t *testing.T) {
		t.Parallel()

		host := &contextmock.VMHostMock{}

		storageContextInstance, err := NewStorageContext(host, nil, elrondReservedTestPrefix)
		require.Equal(t, arwen.ErrNilBlockChainHook, err)
		require.True(t, check.IfNil(storageContextInstance))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsStorageAPICostOptimizationFlagEnabledField: true,
		}
		host := &contextmock.VMHostMock{
			EnableEpochsHandlerField: enableEpochsHandler,
		}
		mockBlockchain := worldmock.NewMockWorld()

		storageContextInstance, err := NewStorageContext(host, mockBlockchain, elrondReservedTestPrefix)
		require.Nil(t, err)
		require.False(t, check.IfNil(storageContextInstance))
	})
}

func TestStorageContext_SetAddress(t *testing.T) {
	t.Parallel()

	addressA := []byte("accountA")
	addressB := []byte("accountB")
	stubOutput := &contextmock.OutputContextStub{}
	accountA := &vmcommon.OutputAccount{
		Address:        addressA,
		Nonce:          0,
		BalanceDelta:   big.NewInt(0),
		Balance:        big.NewInt(0),
		StorageUpdates: make(map[string]*vmcommon.StorageUpdate),
	}
	accountB := &vmcommon.OutputAccount{
		Address:        addressB,
		Nonce:          0,
		BalanceDelta:   big.NewInt(0),
		Balance:        big.NewInt(0),
		StorageUpdates: make(map[string]*vmcommon.StorageUpdate),
	}
	stubOutput.GetOutputAccountCalled = func(address []byte) (*vmcommon.OutputAccount, bool) {
		if bytes.Equal(address, addressA) {
			return accountA, false
		}
		if bytes.Equal(address, addressB) {
			return accountB, false
		}
		return nil, false
	}

	mockRuntime := &contextmock.RuntimeContextMock{}
	mockMetering := &contextmock.MeteringContextMock{}
	mockMetering.SetGasSchedule(config.MakeGasMapForTests())
	mockMetering.BlockGasLimitMock = uint64(15000)
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsStorageAPICostOptimizationFlagEnabledField: true,
	}

	host := &contextmock.VMHostMock{
		OutputContext:            stubOutput,
		MeteringContext:          mockMetering,
		RuntimeContext:           mockRuntime,
		EnableEpochsHandlerField: enableEpochsHandler,
	}
	bcHook := &contextmock.BlockchainHookStub{}

	storageContextInstance, _ := NewStorageContext(host, bcHook, elrondReservedTestPrefix)

	keyA := []byte("keyA")
	valueA := []byte("valueA")

	storageContextInstance.SetAddress(addressA)
	storageStatus, err := storageContextInstance.SetStorage(keyA, valueA)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageAdded, storageStatus)
	require.Equal(t, uint64(len(valueA)), accountA.BytesAddedToStorage)
	require.Equal(t, uint64(0), accountA.BytesDeletedFromStorage)
	foundValueA, _, _ := storageContextInstance.GetStorage(keyA)
	require.Equal(t, valueA, foundValueA)
	require.Len(t, storageContextInstance.GetStorageUpdates(addressA), 1)
	require.Len(t, storageContextInstance.GetStorageUpdates(addressB), 0)

	keyB := []byte("keyB")
	valueB := []byte("valueB")
	storageContextInstance.SetAddress(addressB)
	storageStatus, err = storageContextInstance.SetStorage(keyB, valueB)
	require.Equal(t, uint64(len(valueB)), accountB.BytesAddedToStorage)
	require.Equal(t, uint64(0), accountB.BytesDeletedFromStorage)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageAdded, storageStatus)
	foundValueB, _, _ := storageContextInstance.GetStorage(keyB)
	require.Equal(t, valueB, foundValueB)
	require.Len(t, storageContextInstance.GetStorageUpdates(addressA), 1)
	require.Len(t, storageContextInstance.GetStorageUpdates(addressB), 1)
	foundValueA, _, _ = storageContextInstance.GetStorage(keyA)
	require.Equal(t, []byte(nil), foundValueA)
}

func TestStorageContext_GetStorageUpdates(t *testing.T) {
	t.Parallel()

	mockOutput := &contextmock.OutputContextMock{}
	account := mockOutput.NewVMOutputAccount([]byte("account"))
	mockOutput.OutputAccountMock = account
	mockOutput.OutputAccountIsNew = false

	account.StorageUpdates["update"] = &vmcommon.StorageUpdate{
		Offset: []byte("update"),
		Data:   []byte("some data"),
	}

	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsStorageAPICostOptimizationFlagEnabledField: true,
	}

	host := &contextmock.VMHostMock{
		OutputContext:            mockOutput,
		EnableEpochsHandlerField: enableEpochsHandler,
	}

	mockBlockchainHook := worldmock.NewMockWorld()
	storageContextInstance, _ := NewStorageContext(host, mockBlockchainHook, elrondReservedTestPrefix)

	storageUpdates := storageContextInstance.GetStorageUpdates([]byte("account"))
	require.Equal(t, 1, len(storageUpdates))
	require.Equal(t, []byte("update"), storageUpdates["update"].Offset)
	require.Equal(t, []byte("some data"), storageUpdates["update"].Data)
}

func TestStorageContext_SetStorage(t *testing.T) {
	t.Parallel()

	address := []byte("account")
	mockOutput := &contextmock.OutputContextMock{}
	account := mockOutput.NewVMOutputAccount(address)
	mockOutput.OutputAccountMock = account
	mockOutput.OutputAccountIsNew = false

	mockRuntime := &contextmock.RuntimeContextMock{}
	mockMetering := &contextmock.MeteringContextMock{}
	mockMetering.SetGasSchedule(config.MakeGasMapForTests())
	mockMetering.BlockGasLimitMock = uint64(15000)

	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsStorageAPICostOptimizationFlagEnabledField: true,
	}

	host := &contextmock.VMHostMock{
		OutputContext:            mockOutput,
		MeteringContext:          mockMetering,
		RuntimeContext:           mockRuntime,
		EnableEpochsHandlerField: enableEpochsHandler,
	}
	bcHook := &contextmock.BlockchainHookStub{}
	storageContextInstance, _ := NewStorageContext(host, bcHook, elrondReservedTestPrefix)
	storageContextInstance.SetAddress(address)

	val1 := []byte("value")
	val2 := []byte("newValue")
	val3 := []byte("v")

	key := []byte("key")
	value := val1
	addedBytes := len(value)

	storageStatus, err := storageContextInstance.SetStorage(key, value)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageAdded, storageStatus)
	require.Equal(t, uint64(addedBytes), account.BytesAddedToStorage)
	require.Equal(t, uint64(0), account.BytesDeletedFromStorage)
	foundValue, _, _ := storageContextInstance.GetStorage(key)
	require.Equal(t, value, foundValue)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)

	value = val2
	addedBytes += len(value) - len(val1)

	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageModified, storageStatus)
	require.Equal(t, uint64(addedBytes), account.BytesAddedToStorage)
	require.Equal(t, uint64(0), account.BytesDeletedFromStorage)
	foundValue, _, _ = storageContextInstance.GetStorage(key)
	require.Equal(t, value, foundValue)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)

	value = val2

	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageUnchanged, storageStatus)
	require.Equal(t, uint64(addedBytes), account.BytesAddedToStorage)
	require.Equal(t, uint64(0), account.BytesDeletedFromStorage)
	foundValue, _, _ = storageContextInstance.GetStorage(key)
	require.Equal(t, value, foundValue)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)

	value = val1
	deletedBytes := len(val2) - len(val1)

	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageModified, storageStatus)
	require.Equal(t, uint64(addedBytes), account.BytesAddedToStorage)
	require.Equal(t, uint64(deletedBytes), account.BytesDeletedFromStorage)
	foundValue, _, _ = storageContextInstance.GetStorage(key)
	require.Equal(t, value, foundValue)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)

	value = val3
	deletedBytes += len(val1) - len(val3)

	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageModified, storageStatus)
	require.Equal(t, uint64(addedBytes), account.BytesAddedToStorage)
	require.Equal(t, uint64(deletedBytes), account.BytesDeletedFromStorage)
	foundValue, _, _ = storageContextInstance.GetStorage(key)
	require.Equal(t, value, foundValue)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)

	value = nil
	deletedBytes += len(val3)

	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageDeleted, storageStatus)
	require.Equal(t, uint64(addedBytes), account.BytesAddedToStorage)
	require.Equal(t, uint64(deletedBytes), account.BytesDeletedFromStorage)
	foundValue, _, _ = storageContextInstance.GetStorage(key)
	require.Equal(t, []byte{}, foundValue)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)

	mockRuntime.SetReadOnly(true)
	value = val2
	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Equal(t, err, arwen.ErrCannotWriteOnReadOnly)
	require.Equal(t, arwen.StorageUnchanged, storageStatus)
	foundValue, _, _ = storageContextInstance.GetStorage(key)
	require.Equal(t, []byte{}, foundValue)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)

	mockRuntime.SetReadOnly(false)
	key = []byte("other_key")
	value = []byte("other_value")
	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageAdded, storageStatus)
	foundValue, _, _ = storageContextInstance.GetStorage(key)
	require.Equal(t, value, foundValue)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 2)

	key = []byte("RESERVEDkey")
	value = []byte("doesn't matter")
	_, err = storageContextInstance.SetStorage(key, value)
	require.Equal(t, arwen.ErrStoreElrondReservedKey, err)

	key = []byte("RESERVED")
	value = []byte("doesn't matter")
	_, err = storageContextInstance.SetStorage(key, value)
	require.Equal(t, arwen.ErrStoreElrondReservedKey, err)
}

func TestStorageConext_SetStorage_GasUsage(t *testing.T) {
	address := []byte("account")
	mockOutput := &contextmock.OutputContextMock{}
	account := mockOutput.NewVMOutputAccount(address)
	mockOutput.OutputAccountMock = account
	mockOutput.OutputAccountIsNew = false

	storeCost := 11
	persistCost := 7
	releaseCost := 5

	gasMap := config.MakeGasMapForTests()
	gasMap["BaseOperationCost"]["StorePerByte"] = uint64(storeCost)
	gasMap["BaseOperationCost"]["PersistPerByte"] = uint64(persistCost)
	gasMap["BaseOperationCost"]["ReleasePerByte"] = uint64(releaseCost)

	mockRuntime := &contextmock.RuntimeContextMock{}
	mockMetering := &contextmock.MeteringContextMock{}
	mockMetering.SetGasSchedule(gasMap)
	mockMetering.BlockGasLimitMock = uint64(15000)

	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsStorageAPICostOptimizationFlagEnabledField: true,
	}
	host := &contextmock.VMHostMock{
		OutputContext:            mockOutput,
		MeteringContext:          mockMetering,
		RuntimeContext:           mockRuntime,
		EnableEpochsHandlerField: enableEpochsHandler,
	}
	bcHook := &contextmock.BlockchainHookStub{}

	storageContextInstance, _ := NewStorageContext(host, bcHook, elrondReservedTestPrefix)
	storageContextInstance.SetAddress(address)

	gasProvided := 100
	mockMetering.GasLeftMock = uint64(gasProvided)
	key := []byte("key")

	// Store new value
	value := []byte("value")
	storageStatus, err := storageContextInstance.SetStorage(key, value)
	gasLeft := gasProvided - storeCost*len(value)
	storedValue, _, _ := storageContextInstance.GetStorage(key)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageAdded, storageStatus)
	require.Equal(t, gasLeft, int(mockMetering.GasLeft()))
	require.Equal(t, value, storedValue)

	// Update with longer value
	value2 := []byte("value2")
	mockMetering.GasLeftMock = uint64(gasProvided)
	storageStatus, err = storageContextInstance.SetStorage(key, value2)
	storedValue, _, _ = storageContextInstance.GetStorage(key)
	gasLeft = gasProvided - persistCost*len(value) - storeCost*(len(value2)-len(value))
	require.Nil(t, err)
	require.Equal(t, arwen.StorageModified, storageStatus)
	require.Equal(t, gasLeft, int(mockMetering.GasLeft()))
	require.Equal(t, value2, storedValue)

	// Revert to initial value
	mockMetering.GasLeftMock = uint64(gasProvided)
	storageStatus, err = storageContextInstance.SetStorage(key, value)
	gasLeft = gasProvided - persistCost*len(value)
	gasFreed := releaseCost * (len(value2) - len(value))
	storedValue, _, _ = storageContextInstance.GetStorage(key)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageModified, storageStatus)
	require.Equal(t, gasLeft, int(mockMetering.GasLeft()))
	require.Equal(t, gasFreed, int(mockMetering.GasFreedMock))
	require.Equal(t, value, storedValue)
}

func TestStorageContext_StorageProtection(t *testing.T) {
	address := []byte("account")
	mockOutput := &contextmock.OutputContextMock{}
	account := mockOutput.NewVMOutputAccount(address)
	mockOutput.OutputAccountMock = account
	mockOutput.OutputAccountIsNew = false

	mockRuntime := &contextmock.RuntimeContextMock{}
	mockMetering := &contextmock.MeteringContextMock{}
	mockMetering.SetGasSchedule(config.MakeGasMapForTests())
	mockMetering.BlockGasLimitMock = uint64(15000)

	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsStorageAPICostOptimizationFlagEnabledField: true,
	}

	host := &contextmock.VMHostMock{
		OutputContext:            mockOutput,
		MeteringContext:          mockMetering,
		RuntimeContext:           mockRuntime,
		EnableEpochsHandlerField: enableEpochsHandler,
	}
	bcHook := &contextmock.BlockchainHookStub{}

	storageContextInstance, _ := NewStorageContext(host, bcHook, elrondReservedTestPrefix)
	storageContextInstance.SetAddress(address)

	key := []byte(arwen.ProtectedStoragePrefix + "something")
	value := []byte("data")

	storageStatus, err := storageContextInstance.SetStorage(key, value)
	require.Equal(t, arwen.StorageUnchanged, storageStatus)
	require.True(t, errors.Is(err, arwen.ErrCannotWriteProtectedKey))
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 0)

	storageContextInstance.disableStorageProtection()
	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Nil(t, err)
	require.Equal(t, arwen.StorageAdded, storageStatus)
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)

	storageContextInstance.enableStorageProtection()
	storageStatus, err = storageContextInstance.SetStorage(key, value)
	require.Equal(t, arwen.StorageUnchanged, storageStatus)
	require.True(t, errors.Is(err, arwen.ErrCannotWriteProtectedKey))
	require.Len(t, storageContextInstance.GetStorageUpdates(address), 1)
}

func TestStorageContext_GetStorageFromAddress(t *testing.T) {
	t.Parallel()

	t.Run("blockchain hook errors", func(t *testing.T) {
		t.Parallel()

		scAddress := []byte("account")
		mockOutput := &contextmock.OutputContextMock{}
		account := mockOutput.NewVMOutputAccount(scAddress)
		mockOutput.OutputAccountMock = account
		mockOutput.OutputAccountIsNew = false

		mockRuntime := &contextmock.RuntimeContextMock{}
		mockMetering := &contextmock.MeteringContextMock{}
		mockMetering.SetGasSchedule(config.MakeGasMapForTests())
		mockMetering.BlockGasLimitMock = uint64(15000)

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsStorageAPICostOptimizationFlagEnabledField: true,
		}

		host := &contextmock.VMHostMock{
			OutputContext:            mockOutput,
			MeteringContext:          mockMetering,
			RuntimeContext:           mockRuntime,
			EnableEpochsHandlerField: enableEpochsHandler,
		}

		readable := []byte("readable")
		nonreadable := []byte("nonreadable")

		errTooManyRequests := errors.New("too many requests")
		bcHook := &contextmock.BlockchainHookStub{
			GetUserAccountCalled: func(address []byte) (vmcommon.UserAccountHandler, error) {
				if bytes.Equal(readable, address) {
					return &worldmock.Account{CodeMetadata: []byte{4, 0}}, nil
				}
				if bytes.Equal(nonreadable, address) || bytes.Equal(scAddress, address) {
					return &worldmock.Account{CodeMetadata: []byte{0, 0}}, nil
				}
				return nil, nil
			},
			GetStorageDataCalled: func(accountsAddress []byte, index []byte) ([]byte, uint32, error) {
				return nil, 0, errTooManyRequests
			},
		}

		storageContextInstance, _ := NewStorageContext(host, bcHook, elrondReservedTestPrefix)
		storageContextInstance.SetAddress(scAddress)

		key := []byte("key")
		data, _, err := storageContextInstance.GetStorageFromAddress(scAddress, key)
		require.Nil(t, data)
		assert.Equal(t, errTooManyRequests, err)

		data, _, _ = storageContextInstance.GetStorageFromAddress(readable, key)
		require.Nil(t, data)
		assert.Equal(t, errTooManyRequests, err)

		data, _, _ = storageContextInstance.GetStorageFromAddress(nonreadable, key)
		require.Nil(t, data)
		assert.Equal(t, errTooManyRequests, err)
	})
	t.Run("should work when blockchain hook does not error", func(t *testing.T) {
		t.Parallel()

		scAddress := []byte("account")
		mockOutput := &contextmock.OutputContextMock{}
		account := mockOutput.NewVMOutputAccount(scAddress)
		mockOutput.OutputAccountMock = account
		mockOutput.OutputAccountIsNew = false

		mockRuntime := &contextmock.RuntimeContextMock{}
		mockMetering := &contextmock.MeteringContextMock{}
		mockMetering.SetGasSchedule(config.MakeGasMapForTests())
		mockMetering.BlockGasLimitMock = uint64(15000)

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsStorageAPICostOptimizationFlagEnabledField: true,
		}

		host := &contextmock.VMHostMock{
			OutputContext:            mockOutput,
			MeteringContext:          mockMetering,
			RuntimeContext:           mockRuntime,
			EnableEpochsHandlerField: enableEpochsHandler,
		}

		readable := []byte("readable")
		nonreadable := []byte("nonreadable")
		internalData := []byte("internalData")

		bcHook := &contextmock.BlockchainHookStub{
			GetUserAccountCalled: func(address []byte) (vmcommon.UserAccountHandler, error) {
				if bytes.Equal(readable, address) {
					return &worldmock.Account{CodeMetadata: []byte{4, 0}}, nil
				}
				if bytes.Equal(nonreadable, address) || bytes.Equal(scAddress, address) {
					return &worldmock.Account{CodeMetadata: []byte{0, 0}}, nil
				}
				return nil, nil
			},
			GetStorageDataCalled: func(accountsAddress []byte, index []byte) ([]byte, uint32, error) {
				return internalData, 0, nil
			},
		}

		storageContextInstance, _ := NewStorageContext(host, bcHook, elrondReservedTestPrefix)
		storageContextInstance.SetAddress(scAddress)

		key := []byte("key")
		data, _, _ := storageContextInstance.GetStorageFromAddress(scAddress, key)
		require.Equal(t, data, internalData)

		data, _, _ = storageContextInstance.GetStorageFromAddress(readable, key)
		require.Equal(t, data, internalData)

		data, _, _ = storageContextInstance.GetStorageFromAddress(nonreadable, key)
		require.Nil(t, data)
	})
}

func TestStorageContext_LoadGasStoreGasPerKey(t *testing.T) {
	// TODO
}

func TestStorageContext_StoreGasPerKey(t *testing.T) {
	// TODO
}

func TestStorageContext_PopSetActiveStateIfStackIsEmptyShouldNotPanic(t *testing.T) {
	t.Parallel()
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsStorageAPICostOptimizationFlagEnabledField: true,
	}

	host := &contextmock.VMHostMock{
		EnableEpochsHandlerField: enableEpochsHandler,
	}

	storageContextInstance, _ := NewStorageContext(host, &contextmock.BlockchainHookStub{}, elrondReservedTestPrefix)
	storageContextInstance.PopSetActiveState()

	require.Equal(t, 0, len(storageContextInstance.stateStack))
}

func TestStorageContext_PopDiscardIfStackIsEmptyShouldNotPanic(t *testing.T) {
	t.Parallel()
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsStorageAPICostOptimizationFlagEnabledField: true,
	}

	host := &contextmock.VMHostMock{
		EnableEpochsHandlerField: enableEpochsHandler,
	}

	storageContextInstance, _ := NewStorageContext(host, &contextmock.BlockchainHookStub{}, elrondReservedTestPrefix)
	storageContextInstance.PopDiscard()

	require.Equal(t, 0, len(storageContextInstance.stateStack))
}
