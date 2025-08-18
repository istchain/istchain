package community_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/istchain/istchain/x/community"
	"github.com/istchain/istchain/x/community/keeper"
	"github.com/istchain/istchain/x/community/testutil"
	"github.com/stretchr/testify/suite"
)

func TestABCIPayoutAccumulatedStakingRewards(t *testing.T) {
	testFunc := func(ctx sdk.Context, k keeper.Keeper) {
		community.BeginBlocker(ctx, k)
	}
	suite.Run(t, testutil.NewStakingRewardsTestSuite(testFunc))
}
