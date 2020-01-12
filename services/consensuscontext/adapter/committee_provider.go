package adapter

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type CommitteeProvider interface {
	GetCommittee(ctx context.Context, currentBlockHeight primitives.BlockHeight, randomSeed uint64, maxCommitteeSize uint32) ([]primitives.NodeAddress, error)
}

