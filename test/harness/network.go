package harness

import (
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	testinstrumentation "github.com/orbs-network/orbs-network-go/test/harness/instrumentation"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type AcceptanceTestNetwork interface {
	FlushLog()
	LeaderLoopControl() testinstrumentation.BrakingLoop
	Gossip() gossipAdapter.TamperingTransport
	Leader() services.PublicApi
	Validator() services.PublicApi
	LeaderBp() blockStorageAdapter.InMemoryBlockPersistence
	ValidatorBp() blockStorageAdapter.InMemoryBlockPersistence

	Transfer(gatewayNode services.PublicApi, amount uint64) chan interface{}
	GetBalance(node services.PublicApi) chan uint64
}

type acceptanceTestNetwork struct {
	leader            bootstrap.NodeLogic
	validator         bootstrap.NodeLogic
	leaderLatch       testinstrumentation.Latch
	leaderBp          blockStorageAdapter.InMemoryBlockPersistence
	validatorBp       blockStorageAdapter.InMemoryBlockPersistence
	gossip            gossipAdapter.TamperingTransport
	leaderLoopControl testinstrumentation.BrakingLoop

	log []testinstrumentation.BufferedLog
}

func CreateTestNetwork() AcceptanceTestNetwork {
	leaderConfig := config.NewHardCodedConfig(2, "leader")
	validatorConfig := config.NewHardCodedConfig(2, "validator")

	leaderLog := testinstrumentation.NewBufferedLog("leader")
	leaderLatch := testinstrumentation.NewLatch()
	validatorLog := testinstrumentation.NewBufferedLog("validator")

	leaderLoopControl := testinstrumentation.NewBrakingLoop(leaderLog)

	temperingTransport := gossipAdapter.NewTamperingTransport()
	leaderBp := blockStorageAdapter.NewInMemoryBlockPersistence(leaderConfig)
	validatorBp := blockStorageAdapter.NewInMemoryBlockPersistence(validatorConfig)

	leader := bootstrap.NewNodeLogic(temperingTransport, leaderBp, instrumentation.NewCompositeReporting([]instrumentation.Reporting{leaderLog, leaderLatch}), leaderLoopControl, leaderConfig, true)
	validator := bootstrap.NewNodeLogic(temperingTransport, validatorBp, validatorLog, testinstrumentation.NewBrakingLoop(validatorLog), validatorConfig, false)

	return &acceptanceTestNetwork{
		leader:            leader,
		validator:         validator,
		leaderLatch:       leaderLatch,
		leaderBp:          leaderBp,
		validatorBp:       validatorBp,
		gossip:            temperingTransport,
		leaderLoopControl: leaderLoopControl,

		log: []testinstrumentation.BufferedLog{leaderLog, validatorLog},
	}
}

func (n *acceptanceTestNetwork) FlushLog() {
	for _, l := range n.log {
		l.Flush()
	}
}

func (n *acceptanceTestNetwork) LeaderLoopControl() testinstrumentation.BrakingLoop {
	return n.leaderLoopControl
}

func (n *acceptanceTestNetwork) Gossip() gossipAdapter.TamperingTransport {
	return n.gossip
}

func (n *acceptanceTestNetwork) Leader() services.PublicApi {
	return n.leader.GetPublicApi()
}

func (n *acceptanceTestNetwork) Validator() services.PublicApi {
	return n.validator.GetPublicApi()
}

func (n *acceptanceTestNetwork) LeaderBp() blockStorageAdapter.InMemoryBlockPersistence {
	return n.leaderBp
}

func (n *acceptanceTestNetwork) ValidatorBp() blockStorageAdapter.InMemoryBlockPersistence {
	return n.validatorBp
}

func (n *acceptanceTestNetwork) Transfer(gatewayNode services.PublicApi, amount uint64) chan interface{} {
	ch := make(chan interface{})
	go func() {
		tx := &protocol.SignedTransactionBuilder{Transaction: &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "transfer",
			InputArguments: []*protocol.MethodArgumentBuilder{
				{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: amount},
			},
		}}
		input := &services.SendTransactionInput{ClientRequest: (&client.SendTransactionRequestBuilder{SignedTransaction: tx}).Build()}
		gatewayNode.SendTransaction(input)
		ch <- nil
	}()
	return ch
}

func (n *acceptanceTestNetwork) GetBalance(node services.PublicApi) chan uint64 {
	ch := make(chan uint64)
	go func() {
		cm := &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "getBalance",
		}
		input := &services.CallMethodInput{ClientRequest: (&client.CallMethodRequestBuilder{Transaction: cm}).Build()}
		output, _ := node.CallMethod(input)
		ch <- output.ClientResponse.OutputArgumentsIterator().NextOutputArguments().Uint64Value()
	}()
	return ch
}
