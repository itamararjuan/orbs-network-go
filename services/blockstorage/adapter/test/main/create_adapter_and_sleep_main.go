// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package main

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"os"
	"time"
)

func main() {
	config := &localConfig{dir: os.Args[1], virtualChainId: 42}
	_, release, err := test.NewFilesystemAdapterDriver(log.GetLogger(), config)
	if err != nil {
		os.Exit(1)
	}

	defer release()
	time.Sleep(3 * time.Second) // keep the lock until the second instance tries to lock file
}

type localConfig struct {
	dir            string
	virtualChainId primitives.VirtualChainId
	networkType    protocol.SignerNetworkType
}

func (l *localConfig) VirtualChainId() primitives.VirtualChainId {
	return l.virtualChainId
}

func (l *localConfig) NetworkType() protocol.SignerNetworkType {
	return l.networkType
}

func (l *localConfig) BlockStorageFileSystemDataDir() string {
	return l.dir
}

func (l *localConfig) BlockStorageFileSystemMaxBlockSizeInBytes() uint32 {
	return 1000000000
}
