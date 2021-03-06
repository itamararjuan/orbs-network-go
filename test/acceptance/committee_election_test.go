// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build unsafetests

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeanHelix_CommitTransactionToElected(t *testing.T) {
	NewHarness().
		WithNumNodes(6).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect first 4 out of 6")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 0, []int{0, 1, 2, 3})
			test.RequireSuccess(t, response, "elect first 4 out of 6 failed")

			t.Log("send transaction to one of the elected")

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("make sure it arrived to non-elected")

			network.WaitForTransactionInNodeState(ctx, txHash, 4)
			require.EqualValues(t, 10, token.GetBalance(ctx, 4, 6))

			t.Log("send transaction to one of the non-elected")

			_, txHash = token.Transfer(ctx, 4, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 4)
			require.EqualValues(t, 20, token.GetBalance(ctx, 4, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 4, []int{0, 1, 2, 3})

			t.Log("make sure it arrived to elected")

			network.WaitForTransactionInNodeState(ctx, txHash, 2)
			require.EqualValues(t, 20, token.GetBalance(ctx, 2, 6))

			t.Log("test done, shutting down")

		})
}

func TestLeanHelix_MultipleReElections(t *testing.T) {
	NewHarness().
		WithNumNodes(6).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)
			test.RequireSuccess(t, response, "elect 0,1,2,3 failed")

			t.Log("elect 1,2,3,4")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{1, 2, 3, 4})
			test.RequireSuccess(t, response, "elect 1,2,3,4 failed")

			t.Log("elect 2,3,4,5")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{2, 3, 4, 5})
			test.RequireSuccess(t, response, "elect 2,3,4,5 failed")

			t.Log("send transaction to one of the elected")

			_, txHash := token.Transfer(ctx, 3, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 3)
			require.EqualValues(t, 10, token.GetBalance(ctx, 3, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 3, []int{2, 3, 4, 5})

			t.Log("test done, shutting down")

		})
}

func TestLeanHelix_AllNodesLoseElectionButReturn(t *testing.T) {
	NewHarness().
		WithEmptyBlockTime(100*time.Millisecond).
		WithNumNodes(8).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			test.RequireSuccess(t, response, "elect 0,1,2,3 failed")

			t.Log("send transaction to the first group")

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("elect 4,5,6,7 - entire first group loses")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 4, []int{4, 5, 6, 7})
			test.RequireSuccess(t, response, "elect 4,5,6,7 failed")

			t.Log("send transaction to the first group after loss")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 20, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{4, 5, 6, 7})

			t.Log("elect 0,1,2,3 - first group returns")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			test.RequireSuccess(t, response, "re-elect 0,1,2,3 failed")

			t.Log("send transaction to the first node after return")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 30, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("test done, shutting down")

		})
}

func TestLeanHelix_GrowingElectedAmount(t *testing.T) {
	NewHarness().
		WithNumNodes(7).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			test.RequireSuccess(t, response, "elect 0,1,2,3 failed")

			t.Log("send transaction")

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("elect 0,1,2,3,4,5,6")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3, 4, 5, 6})
			test.RequireSuccess(t, response, "elect 0,1,2,3,4,5,6 failed")

			t.Log("send transaction")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 20, token.GetBalance(ctx, 0, 6))

			network.WaitForTransactionReceiptInTransactionPool(ctx, txHash, 0)
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3, 4, 5, 6})

			t.Log("test done, shutting down")

		})
}

func TestLeanHelix_CommitTransactionWithCommitteeContractTurnedOff(t *testing.T) {
	NewHarness().WithConfigOverride(func(cfg config.OverridableConfig) config.OverridableConfig {
		c, err := cfg.MergeWithFileConfig(`{"consensus-context-committee-using-contract": false}`)
		require.NoError(t, err)
		return c
	}).
		WithNumNodes(6).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect first 4 out of 6")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 0, []int{0, 1, 2, 3})
			test.RequireSuccess(t, response, "elect first 4 out of 6 failed")

			t.Log("send transaction to one of the elected")

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("test done, shutting down")

		})
}

func verifyTxSignersAreFromGroup(t testing.TB, ctx context.Context, api callcontract.CallContractAPI, txHash primitives.Sha256, nodeIndex int, allowedIndexes []int) {
	response := api.GetTransactionReceiptProof(ctx, txHash, nodeIndex)
	signers, err := digest.GetBlockSignersFromReceiptProof(response.PackedProof())
	require.NoError(t, err, "failed getting signers from block proof")
	signerIndexes := testKeys.NodeAddressesForTestsToIndexes(signers)
	t.Logf("signers of txHash %s are %v", txHash, signerIndexes)
	require.Subset(t, allowedIndexes, signerIndexes, "tx signers should be subset of allowed group")
}
