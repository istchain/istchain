package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/istchain/istchain/x/community/keeper"
	"github.com/istchain/istchain/x/community/testutil"
)

func TestKeeperPayoutAccumulatedStakingRewards(t *testing.T) {
	testFunc := func(ctx sdk.Context, k keeper.Keeper) {
		k.PayoutAccumulatedStakingRewards(ctx)
	}
	suite.Run(t, testutil.NewStakingRewardsTestSuite(testFunc))
}
