// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//

package stream

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"sync"
)

var LogTag = log.Service("processor-stream")

type service struct {
	logger log.Logger

	mutex               *sync.RWMutex
	sdkHandler          handlers.ContractSdkCallHandler
	contractsUnderMutex map[primitives.ContractName]string
}

func NewEventStreamProcessor(logger log.Logger) services.Processor {
	return &service{
		logger:              logger.WithTags(LogTag),
		mutex:               &sync.RWMutex{},
		contractsUnderMutex: make(map[primitives.ContractName]string),
	}
}

// runs once on system initialization (called by the virtual machine constructor)
func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.sdkHandler = handler
}

func (s *service) ProcessCall(ctx context.Context, input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	// no need to retrieve code
	// execute

	eventName := input.MethodName.String()
	outputArgs := protocol.ArgumentsArrayEmpty()

	if eventName == "_init" {
		return &services.ProcessCallOutput{
			OutputArgumentArray: outputArgs,
			CallResult:          protocol.EXECUTION_RESULT_SUCCESS,
		}, nil
	}

	args, err := protocol.ArgumentsFromNatives([]interface{}{eventName, input.InputArgumentArray.Raw()})
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArgumentArray: outputArgs,
			CallResult:          protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	_, contractErr := s.sdkHandler.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		PermissionScope: input.CallingPermissionScope,
		ContextId:       input.ContextId,
		OperationName:   sdk.SDK_OPERATION_NAME_EVENTS,
		MethodName:      "emitEvent",
		InputArguments:  args,
	})

	// result
	callResult := protocol.EXECUTION_RESULT_SUCCESS
	if contractErr != nil {
		callResult = protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT
	}
	return &services.ProcessCallOutput{
		OutputArgumentArray: outputArgs,
		CallResult:          callResult,
	}, contractErr
}

func (s *service) GetContractInfo(ctx context.Context, input *services.GetContractInfoInput) (*services.GetContractInfoOutput, error) {
	panic("Not implemented")
}

func (s *service) getContractSdkHandler() handlers.ContractSdkCallHandler {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.sdkHandler
}

func (s *service) getContractFromRepository(contractName primitives.ContractName) string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	panic("not implemented")
}

func (s *service) addContractToRepository(contractName primitives.ContractName, code string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.contractsUnderMutex == nil {
		return
	}
	s.contractsUnderMutex[contractName] = code
}
