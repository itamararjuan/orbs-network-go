// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/servicesync"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"sync"
	"time"
)

const (
	ProtocolVersion = primitives.ProtocolVersion(1)
)

var LogTag = log.Service("block-storage")

type Service struct {
	govnr.TreeSupervisor
	persistence  adapter.BlockPersistence
	stateStorage services.StateStorage
	gossip       gossiptopics.BlockSync
	txPool       services.TransactionPool

	config config.BlockStorageConfig

	logger                  log.Logger
	consensusBlocksHandlers struct {
		sync.RWMutex
		handlers []handlers.ConsensusBlocksHandler
	}

	// lastCommittedBlock state variable is inside adapter.BlockPersistence (GetLastBlock)

	nodeSync *internodesync.BlockSync

	metrics        *metrics
	notifyNodeSync chan struct{}
}

type metrics struct {
	blockHeight       *metric.Gauge
	lastCommittedTime *metric.Gauge
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		blockHeight:       m.NewGauge("BlockStorage.BlockHeight"),
		lastCommittedTime: m.NewGauge("BlockStorage.LastCommitted.TimeNano"),
	}
}

func NewBlockStorage(ctx context.Context, config config.BlockStorageConfig, persistence adapter.BlockPersistence, gossip gossiptopics.BlockSync,
	parentLogger log.Logger, metricFactory metric.Factory, blockPairReceivers []servicesync.BlockPairCommitter) *Service {
	logger := parentLogger.WithTags(LogTag)

	s := &Service{
		persistence:    persistence,
		gossip:         gossip,
		logger:         logger,
		config:         config,
		metrics:        newMetrics(metricFactory),
		notifyNodeSync: make(chan struct{}),
	}

	gossip.RegisterBlockSyncHandler(s)
	s.nodeSync = internodesync.NewBlockSync(ctx, config, gossip, s, logger, metricFactory)

	for _, bpr := range blockPairReceivers {
		s.Supervise(servicesync.NewServiceBlockSync(ctx, logger, persistence, bpr))
	}
	s.Supervise(s.nodeSync)
	s.Supervise(s.startNotifyNodeSync(ctx))

	lastBlock, err := persistence.GetLastBlock()
	if err != nil {
		logger.Error("could not read block from adapter", log.Error(err))
	}
	var height int64
	var lastCommittedTime int64

	if lastBlock != nil {
		height = int64(lastBlock.TransactionsBlock.Header.BlockHeight())
		lastCommittedTime = int64(lastBlock.TransactionsBlock.Header.Timestamp())
	}

	s.metrics.blockHeight.Update(height)
	s.metrics.lastCommittedTime.Update(lastCommittedTime)

	return s
}

func getBlockHeight(block *protocol.BlockPairContainer) primitives.BlockHeight {
	if block == nil {
		return 0
	}
	return block.TransactionsBlock.Header.BlockHeight()
}

func getBlockTimestamp(block *protocol.BlockPairContainer) primitives.TimestampNano {
	if block == nil {
		return 0
	}
	return block.TransactionsBlock.Header.Timestamp()
}

func (s *Service) RegisterConsensusBlocksHandler(handler handlers.ConsensusBlocksHandler) {
	s.appendHandlerUnderLock(handler)
	// update the consensus algo about the latest block we have (for its initialization)
	s.UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(context.TODO())
}

func (s *Service) appendHandlerUnderLock(handler handlers.ConsensusBlocksHandler) {
	s.consensusBlocksHandlers.Lock()
	defer s.consensusBlocksHandlers.Unlock()
	s.consensusBlocksHandlers.handlers = append(s.consensusBlocksHandlers.handlers, handler)
}

func (s *Service) startNotifyNodeSync(ctx context.Context) govnr.ShutdownWaiter {
	return govnr.Forever(ctx, "node sync commit updater", logfields.GovnrErrorer(s.logger), func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.notifyNodeSync:
				s.notifyNodeSyncOfCommittedBlock(ctx)
			}
		}
	})
}

func (s *Service) notifyNodeSyncOfCommittedBlock(ctx context.Context) {
	shortCtx, cancel := context.WithTimeout(ctx, time.Second) // TODO V1 move timeout to configuration
	defer cancel()
	s.nodeSync.HandleBlockCommitted(shortCtx)

}
