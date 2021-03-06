// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//
// +build javascript

package test

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/go-mock"
	config2 "github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/processor/javascript"
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
)

type config struct {
	path string
}

func (c *config) ExperimentalExternalProcessorPluginPath() string {
	return c.path
}

func (c *config) VirtualChainId() primitives.VirtualChainId {
	return 42
}

type harness struct {
	sdkCallHandler *handlers.MockContractSdkCallHandler
	service        services.Processor
}

func newHarness(logger log.Logger, pluginPath string) *harness {

	sdkCallHandler := &handlers.MockContractSdkCallHandler{}

	service := javascript.NewJavaScriptProcessor(logger, &config{
		pluginPath,
	})
	service.RegisterContractSdkCallHandler(sdkCallHandler)

	return &harness{
		sdkCallHandler: sdkCallHandler,
		service:        service,
	}
}

func (h *harness) expectSdkCallMadeWithStateRead(expectedKey []byte, returnValue []byte) {
	stateReadCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == sdk.SDK_OPERATION_NAME_STATE &&
			input.MethodName == "read" &&
			len(input.InputArguments) == 1 &&
			(expectedKey == nil || bytes.Equal(input.InputArguments[0].BytesValue(), expectedKey))
	}

	outputArgs, _ := protocol.ArgumentsFromNatives(builders.VarsToSlice(returnValue)) // err ignored on purpose
	readReturn := &handlers.HandleSdkCallOutput{
		OutputArguments: outputArgs,
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.Any, mock.AnyIf("Contract equals Sdk.State, method equals read and 1 arg matches", stateReadCallMatcher)).Return(readReturn, nil).Times(1)
}

func (h *harness) expectSdkCallMadeWithStateWrite(expectedKey []byte, expectedValue []byte) {
	stateWriteCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == sdk.SDK_OPERATION_NAME_STATE &&
			input.MethodName == "write" &&
			len(input.InputArguments) == 2 &&
			(expectedKey == nil || bytes.Equal(input.InputArguments[0].BytesValue(), expectedKey)) &&
			(expectedValue == nil || bytes.Equal(input.InputArguments[1].BytesValue(), expectedValue))
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.Any, mock.AnyIf("Contract equals Sdk.State, method equals write and 2 args match", stateWriteCallMatcher)).Return(nil, nil).Times(1)
}

func (h *harness) expectSdkCallMadeWithServiceCallMethod(expectedContractName string, expectedMethodName string, expectedArgArray *protocol.ArgumentArray, returnArgArray *protocol.ArgumentArray, returnError error) {
	serviceCallMethodCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == sdk.SDK_OPERATION_NAME_SERVICE &&
			input.MethodName == "callMethod" &&
			len(input.InputArguments) == 3 &&
			input.InputArguments[0].StringValue() == expectedContractName &&
			input.InputArguments[1].StringValue() == expectedMethodName &&
			bytes.Equal(input.InputArguments[2].BytesValue(), expectedArgArray.Raw())
	}

	var returnOutput *handlers.HandleSdkCallOutput
	if returnArgArray != nil {
		outputArgs, _ := protocol.ArgumentsFromNatives(builders.VarsToSlice(returnArgArray.Raw())) // err ignored on purpose
		returnOutput = &handlers.HandleSdkCallOutput{
			OutputArguments: outputArgs,
		}
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.Any, mock.AnyIf("Contract equals Sdk.Service, method equals callMethod and 3 args match", serviceCallMethodCallMatcher)).Return(returnOutput, returnError).Times(1)
}

func (h *harness) expectSdkCallMadeWithAddressGetCaller(returnAddress []byte) {
	addressGetCallerCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == sdk.SDK_OPERATION_NAME_ADDRESS &&
			input.MethodName == "getCallerAddress"
	}

	outputArgs, _ := protocol.ArgumentsFromNatives(builders.VarsToSlice(returnAddress)) // err ignored on purpose
	returnOutput := &handlers.HandleSdkCallOutput{
		OutputArguments: outputArgs,
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.Any, mock.AnyIf("Contract equals Sdk.Address, method equals getCallerAddress and 1 arg match", addressGetCallerCallMatcher)).Return(returnOutput, nil).Times(1)
}

func (h *harness) verifySdkCallMade(t *testing.T) {
	_, err := h.sdkCallHandler.Verify()
	require.NoError(t, err, "sdkCallHandler should be called as expected")
}

func BuildDummyPlugin(src string, target string) {
	root := config2.GetProjectSourceRootPath()
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-tags", "javascript", "-o", DummyPluginPath(target), path.Join(root, src))
	cmd.Dir = root
	cmd.Env = []string{
		"GOPATH=" + getGOPATH(),
		"PATH=" + os.Getenv("PATH"),
		"GO111MODULE=on",
		"HOME=" + os.Getenv("HOME"),
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("failed to compile dummy plugin: %s\n%s", err, string(out)))
	}
}

func RemoveDummyPlugin(target string) {
	os.RemoveAll(DummyPluginPath(target))
}

func DummyPluginPath(target string) string {
	return path.Join(config2.GetProjectSourceRootPath(), target)
}

func getGOPATH() string {
	res := os.Getenv("GOPATH")
	if res == "" {
		return filepath.Join(os.Getenv("HOME"), "go")
	}
	return res
}
