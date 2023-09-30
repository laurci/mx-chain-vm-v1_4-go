package contexts

import (
	"fmt"
	"unicode"

	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-v1_4-go/vmhost"
	"github.com/multiversx/mx-chain-vm-v1_4-go/wasmer"
)

const noArity = -1

// wasmValidator is a validator for WASM SmartContracts
type wasmValidator struct {
	reserved *reservedFunctions
}

// newWASMValidator creates a new WASMValidator
func newWASMValidator(scAPINames vmcommon.FunctionNames, builtInFuncContainer vmcommon.BuiltInFunctionContainer) *wasmValidator {
	return &wasmValidator{
		reserved: NewReservedFunctions(scAPINames, builtInFuncContainer),
	}
}

func (validator *wasmValidator) verifyMemoryDeclaration(instance wasmer.InstanceHandler) error {
	if !instance.HasMemory() {
		return vmhost.ErrMemoryDeclarationMissing
	}

	return nil
}

func (validator *wasmValidator) verifyFunctions(instance wasmer.InstanceHandler) error {
	for functionName := range instance.GetExports() {
		err := validator.verifyValidFunctionName(functionName)
		if err != nil {
			return err
		}

		if functionName != "mx_alloc" {
			err = validator.verifyVoidFunction(instance, functionName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

var protectedFunctions = map[string]bool{
	"internalVMErrors":  true,
	"transferValueOnly": true,
	"writeLog":          true,
	"signalError":       true,
	"completedTxEvent":  true}

func (validator *wasmValidator) verifyProtectedFunctions(instance wasmer.InstanceHandler) error {
	for functionName := range instance.GetExports() {
		_, found := protectedFunctions[functionName]
		if found {
			return vmhost.ErrContractInvalid
		}

	}

	return nil
}

func (validator *wasmValidator) verifyVoidFunction(instance wasmer.InstanceHandler, functionName string) error {
	inArity, err := validator.getInputArity(instance, functionName)
	if err != nil {
		return err
	}

	outArity, err := validator.getOutputArity(instance, functionName)
	if err != nil {
		return err
	}

	isVoid := inArity == 0 && outArity == 0
	if !isVoid {
		return fmt.Errorf("%w: %s", vmhost.ErrFunctionNonvoidSignature, functionName)
	}
	return nil
}

func (validator *wasmValidator) getInputArity(instance wasmer.InstanceHandler, functionName string) (int, error) {
	signature, ok := instance.GetSignature(functionName)
	if !ok {
		return noArity, fmt.Errorf("%w: %s", vmhost.ErrFuncNotFound, functionName)
	}
	return signature.InputArity, nil
}

func (validator *wasmValidator) getOutputArity(instance wasmer.InstanceHandler, functionName string) (int, error) {
	signature, ok := instance.GetSignature(functionName)
	if !ok {
		return noArity, fmt.Errorf("%w: %s", vmhost.ErrFuncNotFound, functionName)
	}
	return signature.OutputArity, nil
}

func (validator *wasmValidator) verifyValidFunctionName(functionName string) error {
	const maxLengthOfFunctionName = 256

	errInvalidName := fmt.Errorf("%w: %s", vmhost.ErrInvalidFunctionName, functionName)

	if len(functionName) == 0 {
		return errInvalidName
	}
	if len(functionName) >= maxLengthOfFunctionName {
		return errInvalidName
	}
	if !isASCIIString(functionName) {
		return errInvalidName
	}
	if validator.reserved.IsReserved(functionName) {
		return errInvalidName
	}

	return nil
}

// TODO: Add more constraints (too loose currently)
func isASCIIString(input string) bool {
	for i := 0; i < len(input); i++ {
		if input[i] > unicode.MaxASCII {
			return false
		}
	}

	return true
}
