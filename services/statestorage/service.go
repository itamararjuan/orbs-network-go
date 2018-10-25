package statestorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"sync"
)

var LogTag = log.Service("state-storage")

type service struct {
	config       config.StateStorageConfig
	blockTracker *synchronization.BlockTracker
	logger       log.BasicLogger

	mutex     sync.RWMutex
	revisions *rollingRevisions
	merkle    *merkle.Forest
}

type merkleCleaner struct {
	f *merkle.Forest
}

func (mc *merkleCleaner) evictRevision(height primitives.BlockHeight, ts primitives.TimestampNano, merkleRoot primitives.MerkleSha256) {
	mc.f.Forget(merkleRoot)
}

func NewStateStorage(config config.StateStorageConfig, persistence adapter.StatePersistence, logger log.BasicLogger) services.StateStorage {

	pHeight, pTs, pRoot, err := persistence.ReadMetadata()
	if err != nil {
		panic(err)
	}
	forest, root, err := buildMerkle(persistence)
	if err != nil {
		panic(err)
	}

	if !root.Equal(pRoot) { // align persisted merkle root with state
		persistence.Write(pHeight, pTs, root, make(adapter.ChainState))
	}

	return &service{
		config:       config,
		blockTracker: synchronization.NewBlockTracker(0, uint16(config.BlockTrackerGraceDistance()), config.BlockTrackerGraceTimeout()),
		logger:       logger.WithTags(LogTag),

		mutex:     sync.RWMutex{},
		merkle:    forest,
		revisions: newRollingRevisions(persistence, int(config.StateStorageHistoryRetentionDistance()), &merkleCleaner{forest}),
	}
}

func buildMerkle(persistence adapter.StatePersistence) (*merkle.Forest, primitives.MerkleSha256, error) {
	forest, root := merkle.NewForest()

	merkleDiffs := make(merkle.MerkleDiffs)
	persistence.Each(func(contract primitives.ContractName, record *protocol.StateRecord) {
		k, v := getMerkleEntry(contract, record.Key(), record.Value())
		merkleDiffs[k] = v
	})

	root, err := forest.Update(root, merkleDiffs)
	return forest, root, err
}

func (s *service) CommitStateDiff(ctx context.Context, input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	if input.ResultsBlockHeader == nil || input.ContractStateDiffs == nil {
		panic("CommitStateDiff received corrupt args")
	}

	commitBlockHeight := input.ResultsBlockHeader.BlockHeight()
	commitTimestamp := input.ResultsBlockHeader.Timestamp()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("trying to commit state diff", log.BlockHeight(commitBlockHeight), log.Int("number-of-state-diffs", len(input.ContractStateDiffs)))

	currentHeight := s.revisions.getCurrentHeight()
	if currentHeight+1 != commitBlockHeight {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: currentHeight + 1}, nil
	}

	// if updating state records fails downstream the merkle tree entries will not bother us
	// TODO use input.resultheader.preexecutuion
	root, err := s.revisions.getRevisionHash(commitBlockHeight - 1)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find previous block merkle root. current block %d", currentHeight)
	}
	newRoot, err := s.merkle.Update(root, filterToMerkleInput(input.ContractStateDiffs))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find new merkle root. current block %d", currentHeight)
	}

	err = s.revisions.addRevision(commitBlockHeight, commitTimestamp, newRoot, inflateChainState(input.ContractStateDiffs))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to write state for block height %d", commitBlockHeight)
	}

	s.blockTracker.IncrementHeight()
	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: commitBlockHeight + 1}, nil
}

func filterToMerkleInput(csd []*protocol.ContractStateDiff) merkle.MerkleDiffs {
	result := make(merkle.MerkleDiffs)
	for _, stateDiffs := range csd {
		contract := stateDiffs.ContractName()
		for i := stateDiffs.StateDiffsIterator(); i.HasNext(); {
			r := i.NextStateDiffs()
			k, v := getMerkleEntry(contract, r.Key(), r.Value())
			result[k] = v
		}
	}
	return result
}

func getMerkleEntry(contract primitives.ContractName, key primitives.Ripmd160Sha256, value []byte) (string, primitives.Sha256) {
	return string(hash.CalcSha256(append([]byte(contract), key...))), hash.CalcSha256(value)
}

func (s *service) ReadKeys(ctx context.Context, input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, errors.Errorf("missing contract name")
	}

	if err := s.blockTracker.WaitForBlock(ctx, input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	currentHeight := s.revisions.getCurrentHeight()
	if input.BlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) <= currentHeight {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, currentHeight, primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
	}

	records := make([]*protocol.StateRecord, 0, len(input.Keys))
	for _, key := range input.Keys {
		record, ok, err := s.revisions.getRevisionRecord(input.BlockHeight, input.ContractName, key.KeyForMap())
		if err != nil {
			return nil, errors.Wrap(err, "persistence layer error")
		}
		if ok {
			records = append(records, record)
		} else { // implicitly return the zero value if key is missing in db
			records = append(records, (&protocol.StateRecordBuilder{Key: key, Value: newZeroValue()}).Build())
		}
	}

	output := &services.ReadKeysOutput{StateRecords: records}
	if len(output.StateRecords) == 0 {
		return output, errors.Errorf("no value found for input key(s)")
	}
	return output, nil
}

func (s *service) GetStateStorageBlockHeight(ctx context.Context, input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    s.revisions.getCurrentHeight(),
		LastCommittedBlockTimestamp: s.revisions.getCurrentTimestamp(),
	}
	return result, nil
}

func (s *service) GetStateHash(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	if err := s.blockTracker.WaitForBlock(ctx, input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	currentHeight := s.revisions.getCurrentHeight()
	if input.BlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) <= currentHeight {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, currentHeight, primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
	}

	value, err := s.revisions.getRevisionHash(input.BlockHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "could not find a merkle root for block height %d", input.BlockHeight)
	}
	output := &services.GetStateHashOutput{StateRootHash: value}

	return output, nil
}

func inflateChainState(csd []*protocol.ContractStateDiff) adapter.ChainState {
	result := make(adapter.ChainState)
	for _, stateDiffs := range csd {
		contract := stateDiffs.ContractName()
		contractMap, ok := result[contract]
		if !ok {
			contractMap = make(map[string]*protocol.StateRecord)
			result[contract] = contractMap
		}
		for i := stateDiffs.StateDiffsIterator(); i.HasNext(); {
			r := i.NextStateDiffs()
			contractMap[r.Key().KeyForMap()] = r
		}
	}
	return result
}

func newZeroValue() []byte {
	return []byte{}
}
