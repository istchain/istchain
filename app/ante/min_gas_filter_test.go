package ante_test

import (
	"strings"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtime "github.com/cometbft/cometbft/types/time"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/istchain/istchain/app"
	"github.com/istchain/istchain/app/ante"
)

func mustParseDecCoins(value string) sdk.DecCoins {
	coins, err := sdk.ParseDecCoins(strings.ReplaceAll(value, ";", ","))
	if err != nil {
		panic(err)
	}

	return coins
}

func TestEvmMinGasFilter(t *testing.T) {
	tApp := app.NewTestApp()
	handler := ante.NewEvmMinGasFilter(tApp.GetEvmKeeper())

	ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})
	tApp.GetEvmKeeper().SetParams(ctx, evmtypes.Params{
		EvmDenom: "aist",
	})

	testCases := []struct {
		name                 string
		minGasPrices         sdk.DecCoins
		expectedMinGasPrices sdk.DecCoins
	}{
		{
			"no min gas prices",
			mustParseDecCoins(""),
			mustParseDecCoins(""),
		},
		{
			"zero uist gas price",
			mustParseDecCoins("0uist"),
			mustParseDecCoins("0uist"),
		},
		{
			"non-zero uist gas price",
			mustParseDecCoins("0.001uist"),
			mustParseDecCoins("0.001uist"),
		},
		{
			"zero uist gas price, min aist price",
			mustParseDecCoins("0uist;100000aist"),
			mustParseDecCoins("0uist"), // aist is removed
		},
		{
			"zero uist gas price, min aist price, other token",
			mustParseDecCoins("0uist;100000aist;0.001other"),
			mustParseDecCoins("0uist;0.001other"), // aist is removed
		},
		{
			"non-zero uist gas price, min aist price",
			mustParseDecCoins("0.25uist;100000aist;0.001other"),
			mustParseDecCoins("0.25uist;0.001other"), // aist is removed
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})

			ctx = ctx.WithMinGasPrices(tc.minGasPrices)
			mmd := MockAnteHandler{}

			_, err := handler.AnteHandle(ctx, nil, false, mmd.AnteHandle)
			require.NoError(t, err)
			require.True(t, mmd.WasCalled)

			assert.NoError(t, mmd.CalledCtx.MinGasPrices().Validate())
			assert.Equal(t, tc.expectedMinGasPrices, mmd.CalledCtx.MinGasPrices())
		})
	}
}
