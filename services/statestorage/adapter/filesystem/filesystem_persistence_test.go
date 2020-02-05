// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package memory

import (
	"context"
	"github.com/orbs-network/lean-helix-go/test"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

type testConfig struct {
}

func (*testConfig) StateStorageHistorySnapshotNum() uint32 {
	panic("implement me")
}

func (*testConfig) StateStorageFileSystemDataDir() string {
	return "."
}

func (*testConfig) BlockTrackerGraceDistance() uint32 {
	panic("implement me")
}

func (*testConfig) BlockTrackerGraceTimeout() time.Duration {
	panic("implement me")
}

func TestReadStateWithNonExistingContractName(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newDriver(ctx)
		_, _, err := d.Read("foo", "")
		require.NoError(t, err, "unexpected error")
	})
}

func TestWriteStateAddAndRemoveKeyFromPersistentStorage(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newDriver(ctx)

		d.writeSingleValueBlock(1, "foo", "foo", "bar")

		record, ok, err := d.Read("foo", "foo")
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, true, ok, "after writing a key it should exist")
		// require.EqualValues(t, "foo", "foo"), "after writing a key/value it should be returned")
		require.EqualValues(t, "bar", record, "after writing a key/value it should be returned")

		d.writeSingleValueBlock(1, "foo", "foo", "")

		_, ok, err = d.Read("foo", "foo")
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, false, ok, "writing zero value to state did not remove key")

	})
}

type driver struct {
	*FilesystemStatePersistence
}

func newDriver(ctx context.Context) *driver {
	removePersistense()
	return &driver{
		NewStatePersistence(ctx, &testConfig{}, metric.NewRegistry()),
	}
}

func (d *driver) writeSingleValueBlock(h primitives.BlockHeight, c, k, v string) error {
	diff := adapter.ChainState{primitives.ContractName(c): {k: []byte(v)}}
	return d.FilesystemStatePersistence.Write(h, 0, []byte{}, []byte{}, diff)
}

func removePersistense() {
	os.RemoveAll("./state.bolt")
}
