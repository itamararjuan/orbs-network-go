package memory

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sort"
)

type Config interface {
	GenesisValidatorNodes() map[string]config.ValidatorNode
}

type committeeTerm struct {
	asOf primitives.BlockHeight
	committee []primitives.NodeAddress
}

type CommitteeProvider struct {
	committees []*committeeTerm
	logger     log.Logger
}

func NewCommitteeProvider(config Config, logger log.Logger) *CommitteeProvider {
	committee := getCommitteeFromConfig(config)
	return  &CommitteeProvider{committees: []*committeeTerm{{0, committee}}, logger :logger}
}


func (cp *CommitteeProvider) GetCommittee(ctx context.Context, requestedBlockHeight primitives.BlockHeight, randomSeed uint64, maxCommitteeSize uint32) ([]primitives.NodeAddress, error) {
	termIndex := len(cp.committees) - 1
	for ; termIndex > 0 && requestedBlockHeight < cp.committees[termIndex].asOf ; termIndex-- {
	}
	return cp.truncateCommitteeToSize(cp.committees[termIndex], maxCommitteeSize), nil
}

func (cp *CommitteeProvider) truncateCommitteeToSize(currentCommittee *committeeTerm, maxCommitteeSize uint32) []primitives.NodeAddress {
	committee := make([]primitives.NodeAddress, 0, maxCommitteeSize)

	for _, nodeAddress := range currentCommittee.committee {
		committee = append(committee, nodeAddress)
		if len(committee) == int(maxCommitteeSize) {
			break
		}
	}
	return committee
}

func getCommitteeFromConfig(config Config) []primitives.NodeAddress {
	allNodes := config.GenesisValidatorNodes()
	var committee []primitives.NodeAddress

	for _, nodeAddress := range allNodes {
		committee = append(committee, nodeAddress.NodeAddress())
	}

	sort.SliceStable(committee, func(i, j int) bool {
		return bytes.Compare(committee[i], committee[j]) > 0
	})
	return committee
}

func (cp *CommitteeProvider) SetCommitteeToTestKeysWithIndices(asOf primitives.BlockHeight, nodeIndices ...int) error {
	if cp.committees[len(cp.committees)-1].asOf >= asOf {
		return errors.Errorf("new committee must have an 'asOf' reference bigger than %d (and not %d)", cp.committees[len(cp.committees)-1].asOf, asOf)
	}
	var committee []primitives.NodeAddress
	for _, committeeIndex := range nodeIndices {
		committee = append(committee, testKeys.EcdsaSecp256K1KeyPairForTests(committeeIndex).NodeAddress())
	}
	cp.committees = append(cp.committees, &committeeTerm{asOf, committee})
	cp.logger.Info("changing committee asof block", logfields.BlockHeight(asOf), log.StringableSlice("committee", committee))
	return nil
}
