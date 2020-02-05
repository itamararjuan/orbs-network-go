// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package memory

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	bolt "go.etcd.io/bbolt"
	"path/filepath"
	"sync"
)

type metrics struct {
	numberOfKeys      *metric.Gauge
	numberOfContracts *metric.Gauge
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		numberOfKeys:      m.NewGauge("StateStoragePersistence.TotalNumberOfKeys.Count"),
		numberOfContracts: m.NewGauge("StateStoragePersistence.TotalNumberOfContracts.Count"),
	}
}

type FilesystemStatePersistence struct {
	metrics    *metrics
	config     config.StateStorageConfig
	mutex      sync.RWMutex
	height     primitives.BlockHeight
	ts         primitives.TimestampNano
	proposer   primitives.NodeAddress
	merkleRoot primitives.Sha256

	db *bolt.DB
}

const STATE_FILENAME = "state"

func NewStatePersistence(ctx context.Context, cfg config.StateStorageConfig, metricFactory metric.Factory) *FilesystemStatePersistence {
	_, merkleRoot := merkle.NewForest()

	db, err := bolt.Open(filepath.Join(cfg.StateStorageFileSystemDataDir(), STATE_FILENAME), 0666, nil)
	if err != nil {
		panic(err)
	}

	// TODO(https://github.com/orbs-network/orbs-network-go/issues/582) - this is our hard coded Genesis block (height 0). Move this to a more dignified place or load from a file
	service := &FilesystemStatePersistence{
		metrics:    newMetrics(metricFactory),
		config:     cfg,
		mutex:      sync.RWMutex{},
		height:     0,
		ts:         0,
		proposer:   []byte{},
		merkleRoot: merkleRoot,
		db:         db,
	}
	go service.closeAutomatically(ctx) // FIXME use govnr
	return service
}

// FIXME report properly
func (sp *FilesystemStatePersistence) reportSize() {
	//nContracts := 0
	//nKeys := 0
	//for _, records := range sp.fullState {
	//	nContracts++
	//	nKeys = nKeys + len(records)
	//}
	//sp.metrics.numberOfKeys.Update(int64(nKeys))
	//sp.metrics.numberOfContracts.Update(int64(nContracts))
}

func (sp *FilesystemStatePersistence) Write(height primitives.BlockHeight, ts primitives.TimestampNano, proposer primitives.NodeAddress, root primitives.Sha256, diff adapter.ChainState) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.height = height
	sp.proposer = proposer
	// TOOD noam why not ts ?
	sp.merkleRoot = root

	for contract, records := range diff {
		for key, value := range records {
			if err := sp._writeOneRecord(primitives.ContractName(contract), key, value); err != nil {
				return err
			}
		}
	}
	sp.reportSize()
	return nil
}

func (sp *FilesystemStatePersistence) _writeOneRecord(c primitives.ContractName, key string, value []byte) error {
	return sp.db.Update(func(tx *bolt.Tx) error {
		contractNameAsBytes := []byte(c)
		keyAsBytes := []byte(key)

		records, err := tx.CreateBucketIfNotExists(contractNameAsBytes)
		if err != nil {
			return err
		}

		if isZeroValue(value) {
			return records.Delete(keyAsBytes)
		}

		return records.Put(keyAsBytes, value)
	})
}

func (sp *FilesystemStatePersistence) Read(contract primitives.ContractName, key string) (record []byte, found bool, err error) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	err = sp.db.View(func(tx *bolt.Tx) error {
		contractNameAsBytes := []byte(contract)
		keyAsBytes := []byte(key)

		records := tx.Bucket(contractNameAsBytes)
		if records == nil {
			return nil
		}

		record = records.Get(keyAsBytes)
		found = record != nil

		return nil
	})

	return
}

func (sp *FilesystemStatePersistence) ReadMetadata() (primitives.BlockHeight, primitives.TimestampNano, primitives.NodeAddress, primitives.Sha256, error) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	return sp.height, sp.ts, sp.proposer, sp.merkleRoot, nil
}

func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}

// FIXME needs further considerations
func (sp *FilesystemStatePersistence) closeAutomatically(ctx context.Context) {
	select {
	case <-ctx.Done():
		if err := sp.db.Close(); err != nil {
			panic(err)
		}
	}
}
