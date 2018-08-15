package instrumentation_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

const (
	TransactionFlow     = "TransactionFlow"
	TransactionAccepted = "Transaction accepted"
)

func captureStdout(f func(writer io.Writer)) string {
	r, w, _ := os.Pipe()

	f(w)

	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func parseOutput(input string) map[string]interface{} {
	jsonMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(input), &jsonMap)
	return jsonMap
}

func TestSimpleLogger(t *testing.T) {
	RegisterTestingT(t)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := instrumentation.GetLogger(instrumentation.Node("node1"), instrumentation.Service("public-api")).WithOutput(instrumentation.Output(writer))
		serviceLogger.Info("Service initialized")
	})

	fmt.Println(stdout)
	jsonMap := parseOutput(stdout)

	Expect(jsonMap["level"]).To(Equal("info"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("instrumentation_test.TestSimpleLogger.func1"))
	Expect(jsonMap["message"]).To(Equal("Service initialized"))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
}

func TestNestedLogger(t *testing.T) {
	RegisterTestingT(t)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := instrumentation.GetLogger(instrumentation.Node("node1"), instrumentation.Service("public-api")).WithOutput(instrumentation.Output(writer))
		txId := instrumentation.String("txId", "1234567")
		txFlowLogger := serviceLogger.For(instrumentation.String("flow", TransactionFlow))
		txFlowLogger.Info(TransactionAccepted, txId, instrumentation.Bytes("payload", []byte{1, 2, 3, 99}))
	})

	fmt.Println(stdout)
	jsonMap := parseOutput(stdout)

	Expect(jsonMap["level"]).To(Equal("info"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("instrumentation_test.TestNestedLogger.func1"))
	Expect(jsonMap["message"]).To(Equal(TransactionAccepted))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
	Expect(jsonMap["txId"]).To(Equal("1234567"))
	Expect(jsonMap["flow"]).To(Equal(TransactionFlow))
	Expect(jsonMap["payload"]).To(Equal("MlZmV0E="))
}

func TestStringableSlice(t *testing.T) {
	RegisterTestingT(t)

	var receipts []*protocol.TransactionReceipt

	receipts = append(receipts, builders.TransactionReceipt().Build())
	receipts = append(receipts, builders.TransactionReceipt().Build())

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := instrumentation.GetLogger(instrumentation.Node("node1"), instrumentation.Service("public-api")).WithOutput(instrumentation.Output(writer))
		serviceLogger.Info("StringableSlice test", instrumentation.StringableSlice("a-collection", receipts))
	})

	fmt.Println(stdout)
	jsonMap := parseOutput(stdout)

	Expect(jsonMap["level"]).To(Equal("info"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("instrumentation_test.TestStringableSlice.func1"))
	Expect(jsonMap["message"]).To(Equal("StringableSlice test"))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
	Expect(jsonMap["a-collection"]).ToNot(Equal("[]"))

	Expect(jsonMap["a-collection"]).To(Equal([]interface{}{
		"{Txhash:736f6d652d74782d68617368,ExecutionResult:EXECUTION_RESULT_SUCCESS,OutputArguments:[],}",
		"{Txhash:736f6d652d74782d68617368,ExecutionResult:EXECUTION_RESULT_SUCCESS,OutputArguments:[],}",
	}))
}

func TestStringableSliceCustomFormat(t *testing.T) {
	RegisterTestingT(t)

	var transactions []*protocol.SignedTransaction

	transactions = append(transactions, builders.TransferTransaction().Build())
	transactions = append(transactions, builders.TransferTransaction().Build())
	transactions = append(transactions, builders.TransferTransaction().Build())
	transactions = append(transactions, builders.TransferTransaction().Build())

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := instrumentation.GetLogger(instrumentation.Node("node1"), instrumentation.Service("public-api")).WithOutput(instrumentation.Output(writer).WithFormatter(instrumentation.NewHumanReadableFormatter()))
		serviceLogger.Info("StringableSlice HR test", instrumentation.StringableSlice("a-collection", transactions))
	})

	fmt.Println(stdout)

	Expect(stdout).To(HavePrefix("info"))
	Expect(stdout).To(ContainSubstring("StringableSlice HR test"))
	Expect(stdout).To(ContainSubstring("node=node1"))
	Expect(stdout).To(ContainSubstring("service=public-api"))
	Expect(stdout).To(ContainSubstring("a-collection=["))
	Expect(stdout).To(ContainSubstring("{Transaction:{ProtocolVersion:1,"))
	Expect(stdout).To(ContainSubstring("function=instrumentation_test.TestStringableSliceCustomFormat.func1"))
	Expect(stdout).To(ContainSubstring("source="))
	Expect(stdout).To(ContainSubstring("instrumentation/basic_logger_test.go"))

}

func TestMeter(t *testing.T) {
	RegisterTestingT(t)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := instrumentation.GetLogger(instrumentation.Node("node1"), instrumentation.Service("public-api")).WithOutput(instrumentation.Output(writer))
		txId := instrumentation.String("txId", "1234567")
		txFlowLogger := serviceLogger.For(instrumentation.String("flow", TransactionFlow))
		meter := txFlowLogger.Meter("tx-process-time", txId)
		defer meter.Done()

		time.Sleep(1 * time.Millisecond)
	})

	fmt.Println(stdout)

	jsonMap := parseOutput(stdout)

	Expect(jsonMap["level"]).To(Equal("metric"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("instrumentation_test.TestMeter.func1"))
	Expect(jsonMap["message"]).To(Equal("Metric recorded"))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
	Expect(jsonMap["metric"]).To(Equal("public-api-TransactionFlow-tx-process-time"))
	Expect(jsonMap["txId"]).To(Equal("1234567"))
	Expect(jsonMap["flow"]).To(Equal(TransactionFlow))
	Expect(jsonMap["process-time"]).NotTo(BeNil())
}

func TestCustomLogFormatter(t *testing.T) {
	RegisterTestingT(t)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := instrumentation.GetLogger(instrumentation.Node("node1"), instrumentation.Service("public-api")).WithOutput(instrumentation.Output(writer).WithFormatter(instrumentation.NewHumanReadableFormatter()))
		serviceLogger.Info("Service initialized", instrumentation.Int("some-int-value", 12), instrumentation.BlockHeight(primitives.BlockHeight(9999)), instrumentation.Bytes("bytes", []byte{2, 3, 99}), instrumentation.Stringable("vchainId", primitives.VirtualChainId(123)))
	})

	fmt.Println(stdout)

	Expect(stdout).To(HavePrefix("info"))
	Expect(stdout).To(ContainSubstring("Service initialized"))
	Expect(stdout).To(ContainSubstring("node=node1"))
	Expect(stdout).To(ContainSubstring("service=public-api"))
	Expect(stdout).To(ContainSubstring("block-height=270f"))
	Expect(stdout).To(ContainSubstring("vchainId=7b"))
	Expect(stdout).To(ContainSubstring("bytes=gDp"))
	Expect(stdout).To(ContainSubstring("some-int-value=12"))
	Expect(stdout).To(ContainSubstring("function=instrumentation_test.TestCustomLogFormatter.func1"))
	Expect(stdout).To(ContainSubstring("source="))
	Expect(stdout).To(ContainSubstring("instrumentation/basic_logger_test.go"))
}

func TestMultipleOutputs(t *testing.T) {
	RegisterTestingT(t)

	filename := "/tmp/test-multiple-outputs"
	os.RemoveAll(filename)

	fileOutput, _ := os.Create(filename)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := instrumentation.GetLogger(instrumentation.Node("node1"), instrumentation.Service("public-api")).WithOutput(instrumentation.Output(writer), instrumentation.Output(fileOutput))
		serviceLogger.Info("Service initialized")
	})

	rawFile, _ := ioutil.ReadFile(filename)
	fileContents := string(rawFile)

	fmt.Println(fileContents)

	checkOutput(stdout)
	checkOutput(fileContents)
}

func TestMultipleOutputsForMemoryViolationByHumanReadable(t *testing.T) {
	RegisterTestingT(t)

	filename := "/tmp/test-multiple-outputs"
	os.RemoveAll(filename)

	fileOutput, _ := os.Create(filename)

	Expect(func() {
		captureStdout(func(writer io.Writer) {
			serviceLogger := instrumentation.GetLogger(instrumentation.Node("node1"), instrumentation.Service("public-api")).WithOutput(instrumentation.Output(writer).WithFormatter(instrumentation.NewHumanReadableFormatter()), instrumentation.Output(fileOutput))
			serviceLogger.Info("Service initialized")
		})
	}).NotTo(Panic())
}

func checkOutput(output string) {
	jsonMap := parseOutput(output)

	Expect(jsonMap["level"]).To(Equal("info"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("instrumentation_test.TestMultipleOutputs.func1"))
	Expect(jsonMap["message"]).To(Equal("Service initialized"))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
}
