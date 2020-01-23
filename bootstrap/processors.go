// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//
// +build !javascript

package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/processor/stream"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

func addExtraProcessors(processors map[protocol.ProcessorType]services.Processor, nodeConfig config.NodeConfig, logger log.Logger) {
	processors[protocol.PROCESSOR_TYPE_JAVASCRIPT] = stream.NewEventStreamProcessor(logger)
}
