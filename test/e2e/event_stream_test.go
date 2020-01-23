package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestEventStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := NewAppHarness()
		lt := time.Now()
		PrintTestTime(t, "started", &lt)

		h.WaitUntilTransactionPoolIsReady(t)
		PrintTestTime(t, "first block committed", &lt)

		PrintTestTime(t, "send deploy - start", &lt)

		contractName := fmt.Sprintf("SomeEvents%d", time.Now().UnixNano())
		deplotmentResponse, _, err := h.SendDeployTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), contractName, orbs.PROCESSOR_TYPE_JAVASCRIPT, nil)
		require.NoError(t, err)
		require.Empty(t, deplotmentResponse.OutputEvents)

		_, txId, err := h.SendTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(),
			contractName, "SomeEvent", "Nicolas Cage", "Vampire's Kill", uint64(1989))
		require.NoError(t, err)

		txStatus, err := h.GetTransactionStatus(txId)
		require.NoError(t, err)
		require.NotEmpty(t, txStatus.OutputEvents)

		require.EqualValues(t, &codec.Event{
			ContractName: contractName,
			EventName:    "SomeEvent",
			Arguments:    []interface{}{"Nicolas Cage", "Vampire's Kill", uint64(1989)},
		}, txStatus.OutputEvents[0])
	})
}
