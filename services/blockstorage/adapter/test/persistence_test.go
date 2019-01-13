package test

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersistenceAdapter_CanAccessBlocksOutOfOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	ctrlRand := test.NewControlledRand(t)
	blocks := builders.RandomizedBlockChain(50, ctrlRand)

	conf := newTempFileConfig()
	defer conf.cleanDir()

	adapter1, close1, err := NewFilesystemAdapterDriver(conf)
	require.NoError(t, err)

	for _, block := range blocks { // write some blocks
		err = adapter1.WriteNextBlock(block)
		require.NoError(t, err)
	}

	requireCanReadAllBlocksInRandomOrder(t, adapter1, blocks, ctrlRand)
	close1()

	adapter2, close2, err := NewFilesystemAdapterDriver(conf)
	require.NoError(t, err)

	requireCanReadAllBlocksInRandomOrder(t, adapter2, blocks, ctrlRand)
	close2()
}

func requireCanReadAllBlocksInRandomOrder(t *testing.T, adapter adapter.BlockPersistence, blocks []*protocol.BlockPairContainer, ctrlRand *test.ControlledRand) {
	for _, i := range ctrlRand.Perm(len(blocks)) { // read each block out of order
		h := primitives.BlockHeight(i + 1)
		block, err := readOneBlock(adapter, h)
		test.RequireCmpEqual(t, blocks[i], block, "expected to succeed in reading block at height %v", h)
		t.Logf("successfully read block height %v", i+1)
		require.NoError(t, err)
	}
}

func readOneBlock(fsa adapter.BlockPersistence, h primitives.BlockHeight) (*protocol.BlockPairContainer, error) {
	var block *protocol.BlockPairContainer
	err := fsa.ScanBlocks(h, 1, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
		block = page[0]
		return false
	})
	return block, err
}