package _manual

import (
	"context"
	"fmt"
	. "github.com/orbs-network/orbs-network-go/services/statestorage/test"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"math"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

const AVG_TX = 100
const TX_COUNT_SIX_MONTHS_AT_AVG_TPX int = 6 * 30 * 24 * 60 * 60 * AVG_TX
const MAX_BLOCK_SIZE = 200
const USERS = 1000000


func TestSimulateStateInitFlowForSixMonthsAt100Tps(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)

		// generate User keys
		userKeys := make([][]byte, USERS)
		for i := range userKeys {
			userKeys[i] = make([]byte, 32)
			rand.Read(userKeys[i])
		}

		var txCount int
		var commits int
		var createInputDuration time.Duration
		var commitDuration time.Duration
		start := time.Now()
		tickStart := start
		var nextBlockHeight primitives.BlockHeight = 1
		for txCount < TX_COUNT_SIX_MONTHS_AT_AVG_TPX { // create input for current simulated block
			generationStart := time.Now()
			commitTxs, commit := getRandomCommit(userKeys, nextBlockHeight,"someContract")

			createInputDuration += time.Now().Sub(generationStart)
			txCount += commitTxs

			// commit state
			output, err := d.CommitStateDiff(ctx, commit)
			require.NoError(t, err)
			nextBlockHeight = output.NextDesiredBlockHeight


			// print progress every 100000 commits
			if commits++; commits % 100 == 0 {
				ms := runtime.MemStats{}
				runtime.ReadMemStats(&ms)
				elapsedTick := time.Now().Sub(tickStart)
				elapsed := time.Now().Sub(start)
				commitDuration = elapsed - createInputDuration
				t.Logf("tick: %v, commit: %v, elapsed: %v, progress: %d, HeapSys: %dMB, HeapAlloc: %dMB\n", elapsedTick, commitDuration, elapsed, 100*txCount/TX_COUNT_SIX_MONTHS_AT_AVG_TPX, ms.HeapSys/ (1024 * 1024), ms.HeapAlloc/ (1024 * 1024))
				tickStart = time.Now()
			}
		}

		// print summary
		commitDuration = time.Now().Sub(start) - createInputDuration
		t.Logf("Wrote    %v transactions in %v blocks to state for %v users in %v\f", txCount, commits, USERS, time.Now().Sub(start))

		require.WithinDuration(t, start, start.Add(commitDuration), time.Minute)

		// test state volume
		var nonZeroBalanceCount int
		for i := range userKeys {
			h, _, _ := d.GetBlockHeightAndTimestamp(ctx)
			value, _ := d.ReadSingleKeyFromRevision(ctx, h,"someContract", string(userKeys[i]))
			if len(value) != 0 {
				nonZeroBalanceCount++
			}
		}
		require.True(t, nonZeroBalanceCount >= int(math.Min(float64(TX_COUNT_SIX_MONTHS_AT_AVG_TPX / 4), USERS)))

		// test memory consumption
		runtime.GC(); runtime.GC(); runtime.GC(); runtime.GC()
		ms := runtime.MemStats{}
		runtime.ReadMemStats(&ms)
		require.True(t, ms.HeapAlloc < 2 * 1024 * 1024 * 1024) // using less than 2GB
	})
}

func getRandomCommit(userKeys [][]byte, height primitives.BlockHeight, contract string) (int, *services.CommitStateDiffInput) {
	blockSize := rand.Int() % MAX_BLOCK_SIZE
	blockDiff := builders.ContractStateDiff().WithContractName(contract)
	for i := 0; i < blockSize; i++ {
		randUser := userKeys[rand.Int()%len(userKeys)]
		randBalance := fmt.Sprintf("%x"+"", rand.Uint64())
		blockDiff.WithStringRecord(string(randUser), randBalance)
	}
	commit := CommitStateDiff().WithBlockHeight(int(height)).WithDiff(blockDiff.Build()).Build()
	return blockSize, commit
}
