package memory

import (
	"context"
	"github.com/orbs-network/lean-helix-go/test"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMemoryCommittee_GetCommitteeWhenOnlyOneTerm(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		cp := newProvider(testKeys.NodeAddressesForTests())

		committee, err := cp.GetCommittee(ctx, 0, 0, 4 )
		require.NoError(t, err)
		require.Len(t, committee, 4, "wrong size of committee")
		require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

		committee, err = cp.GetCommittee(ctx, 10, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")
	})
}

func TestMemoryCommittee_GetCommitteeAfterAnUpdateExists(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		cp := newProvider(testKeys.NodeAddressesForTests())
		termChangeHeight := primitives.BlockHeight(10)
		cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1,2,3,4)

		committee, err := cp.GetCommittee(ctx, termChangeHeight-1, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

		committee, err = cp.GetCommittee(ctx, termChangeHeight, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

		committee, err = cp.GetCommittee(ctx, termChangeHeight+1, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
	})
}

func TestMemoryCommittee_GetCommitteeWhenTwoChangesOneAfterOther(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		cp := newProvider(testKeys.NodeAddressesForTests())
		termChangeHeight := primitives.BlockHeight(10)
		cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1,2,3,4)
		cp.SetCommitteeToTestKeysWithIndices(termChangeHeight+1, 5,6,7,8)

		committee, err := cp.GetCommittee(ctx, termChangeHeight, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

		committee, err = cp.GetCommittee(ctx, termChangeHeight+1, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

		committee, err = cp.GetCommittee(ctx, termChangeHeight+2, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
	})
}

func TestMemoryCommittee_GetCommitteeWhenTwoChangesClose(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		cp := newProvider(testKeys.NodeAddressesForTests())
		termChangeHeight := primitives.BlockHeight(10)
		cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1,2,3,4)
		cp.SetCommitteeToTestKeysWithIndices(termChangeHeight+2, 5,6,7,8)

		committee, err := cp.GetCommittee(ctx, termChangeHeight, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

		committee, err = cp.GetCommittee(ctx, termChangeHeight+1, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

		committee, err = cp.GetCommittee(ctx, termChangeHeight+2, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

		committee, err = cp.GetCommittee(ctx, termChangeHeight+3, 0, 4 )
		require.NoError(t, err)
		require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
	})
}

func TestMemoryCommittee_PreventDoubleCommitteeOnSameBlock(t *testing.T) {
	cp := newProvider(testKeys.NodeAddressesForTests())
	termChangeHeight := primitives.BlockHeight(10)
	err := cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1,2,3,4)
	require.NoError(t, err)

	err = cp.SetCommitteeToTestKeysWithIndices(termChangeHeight - 1, 1,2,3,4)
	require.Error(t, err, "must fail on smaller")

	err = cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1,2,3,4)
	require.Error(t, err, "must fail on equal")
}

func newProvider(committee []primitives.NodeAddress) *CommitteeProvider {
	return  &CommitteeProvider{committees: []*committeeTerm{{0, committee}}}
}
