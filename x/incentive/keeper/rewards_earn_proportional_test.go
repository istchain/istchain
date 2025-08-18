package keeper_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/istchain/istchain/x/incentive/keeper"
	"github.com/istchain/istchain/x/incentive/types"
	"github.com/stretchr/testify/require"
)

func TestGetProportionalRewardPeriod(t *testing.T) {
	tests := []struct {
		name                  string
		giveRewardPeriod      types.MultiRewardPeriod
		giveTotalBkavaSupply  sdkmath.Int
		giveSingleBkavaSupply sdkmath.Int
		wantRewardsPerSecond  sdk.DecCoins
	}{
		{
			"full amount",
			types.NewMultiRewardPeriod(
				true,
				"",
				time.Time{},
				time.Time{},
				cs(c("uist", 100), c("hard", 200)),
			),
			i(100),
			i(100),
			toDcs(c("uist", 100), c("hard", 200)),
		},
		{
			"3/4 amount",
			types.NewMultiRewardPeriod(
				true,
				"",
				time.Time{},
				time.Time{},
				cs(c("uist", 100), c("hard", 200)),
			),
			i(10_000000),
			i(7_500000),
			toDcs(c("uist", 75), c("hard", 150)),
		},
		{
			"half amount",
			types.NewMultiRewardPeriod(
				true,
				"",
				time.Time{},
				time.Time{},
				cs(c("uist", 100), c("hard", 200)),
			),
			i(100),
			i(50),
			toDcs(c("uist", 50), c("hard", 100)),
		},
		{
			"under 1 unit",
			types.NewMultiRewardPeriod(
				true,
				"",
				time.Time{},
				time.Time{},
				cs(c("uist", 100), c("hard", 200)),
			),
			i(1000), // total bist
			i(1),    // bist supply of this specific vault
			dcs(dc("uist", "0.1"), dc("hard", "0.2")), // rewards per second rounded to 0 if under 1uist/1hard
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewardsPerSecond := keeper.GetProportionalRewardsPerSecond(
				tt.giveRewardPeriod,
				tt.giveTotalBkavaSupply,
				tt.giveSingleBkavaSupply,
			)

			require.Equal(t, tt.wantRewardsPerSecond, rewardsPerSecond)
		})
	}
}
