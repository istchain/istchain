package community_test

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/istchain/istchain/app"
	"github.com/istchain/istchain/x/community/types"
)

func TestItCreatesModuleAccountOnInitBlock(t *testing.T) {
	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, tmproto.Header{Height: 1})
	tApp.InitializeFromGenesisStates()

	accKeeper := tApp.GetAccountKeeper()
	acc := accKeeper.GetAccount(ctx, authtypes.NewModuleAddress(types.ModuleName))
	require.NotNil(t, acc)
}
