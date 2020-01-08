package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

type CommitteeProvier interface {
	GetCommittee(ctx context.Context, currentBlockHeight primitives.BlockHeight, randomSeed uint64, maxCommitteeSize uint32) ([]primitives.NodeAddress, error)
}

type Config interface {
	ConsensusContextCommitteeUsingContract() bool
	GenesisValidatorNodes() map[string]config.ValidatorNode
	LeanHelixConsensusMinimumCommitteeSize() uint32 // TODO POS2 should this really be here
}

type PosV1CommitteeProvider struct {
	config         Config
	virtualMachine services.VirtualMachine
	logger         log.Logger
}

func NewPosV1CommitteeProvider(config Config, logger log.Logger, vm services.VirtualMachine) *PosV1CommitteeProvider {
	return &PosV1CommitteeProvider{config:config, logger: logger.WithTags(log.String("adapter", "PosV1CommitteeProvider")), virtualMachine: vm}
}

func (s *PosV1CommitteeProvider) GetCommittee(ctx context.Context, currentBlockHeight primitives.BlockHeight, randomSeed uint64, maxCommitteeSize uint32) ([]primitives.NodeAddress, error) {
	var committee []primitives.NodeAddress
	var err error
	if s.config.ConsensusContextCommitteeUsingContract() {
		committee, err = s.generateCommitteeUsingContract(ctx,  currentBlockHeight, maxCommitteeSize)
		if err != nil {
			return nil, err
		}
	} else {
		committee, err = s.generateCommitteeUsingConsensus(ctx,  currentBlockHeight, randomSeed, maxCommitteeSize)
		if err != nil {
			return nil, err
		}
	}
	return committee, nil
}
