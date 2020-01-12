package posv1

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

type Config interface {
	ConsensusContextCommitteeUsingContract() bool
	GenesisValidatorNodes() map[string]config.ValidatorNode
	LeanHelixConsensusMinimumCommitteeSize() uint32 // TODO POS2 should this really be here
}

type CommitteeProvider struct {
	config         Config
	virtualMachine services.VirtualMachine
	logger         log.Logger
}

func NewCommitteeProvider(config Config, logger log.Logger,) *CommitteeProvider {
	return &CommitteeProvider{config: config, logger: logger.WithTags(log.String("adapter", "PosV1CommitteeProvider"))}
}

func (s *CommitteeProvider) RegisterContractCaller(vm services.VirtualMachine) {
	s.virtualMachine = vm
}

func (s *CommitteeProvider) GetCommittee(ctx context.Context, currentBlockHeight primitives.BlockHeight, randomSeed uint64, maxCommitteeSize uint32) ([]primitives.NodeAddress, error) {
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
