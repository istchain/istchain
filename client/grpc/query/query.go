package query

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	auctiontypes "github.com/istchain/istchain/x/auction/types"
	bep3types "github.com/istchain/istchain/x/bep3/types"
	cdptypes "github.com/istchain/istchain/x/cdp/types"
	committeetypes "github.com/istchain/istchain/x/committee/types"
	communitytypes "github.com/istchain/istchain/x/community/types"
	earntypes "github.com/istchain/istchain/x/earn/types"
	evmutiltypes "github.com/istchain/istchain/x/evmutil/types"
	hardtypes "github.com/istchain/istchain/x/hard/types"
	incentivetypes "github.com/istchain/istchain/x/incentive/types"
	issuancetypes "github.com/istchain/istchain/x/issuance/types"
	istdisttypes "github.com/istchain/istchain/x/istdist/types"
	liquidtypes "github.com/istchain/istchain/x/liquid/types"
	pricefeedtypes "github.com/istchain/istchain/x/pricefeed/types"
	savingstypes "github.com/istchain/istchain/x/savings/types"
	swaptypes "github.com/istchain/istchain/x/swap/types"
)

// QueryClient is a wrapper with all Cosmos and IstChain grpc query clients
type QueryClient struct {
	// cosmos-sdk query clients

	Tm           tmservice.ServiceClient
	Tx           txtypes.ServiceClient
	Auth         authtypes.QueryClient
	Authz        authz.QueryClient
	Bank         banktypes.QueryClient
	Distribution disttypes.QueryClient
	Evidence     evidencetypes.QueryClient
	Gov          govv1types.QueryClient
	GovBeta      govv1beta1types.QueryClient
	Mint         minttypes.QueryClient
	Params       paramstypes.QueryClient
	Slashing     slashingtypes.QueryClient
	Staking      stakingtypes.QueryClient
	Upgrade      upgradetypes.QueryClient
	Consensus    consensustypes.QueryClient

	// 3rd party query clients

	Evm         evmtypes.QueryClient
	Feemarket   feemarkettypes.QueryClient
	IbcClient   ibcclienttypes.QueryClient
	IbcChannel  ibcchanneltypes.QueryClient
	IbcTransfer ibctransfertypes.QueryClient

	// istchain module query clients

	Auction   auctiontypes.QueryClient
	Bep3      bep3types.QueryClient
	Cdp       cdptypes.QueryClient
	Committee committeetypes.QueryClient
	Community communitytypes.QueryClient
	Earn      earntypes.QueryClient
	Evmutil   evmutiltypes.QueryClient
	Hard      hardtypes.QueryClient
	Incentive incentivetypes.QueryClient
	Issuance  issuancetypes.QueryClient
	Istdist   istdisttypes.QueryClient
	Liquid    liquidtypes.QueryClient
	Pricefeed pricefeedtypes.QueryClient
	Savings   savingstypes.QueryClient
	Swap      swaptypes.QueryClient
}

// NewQueryClient creates a new QueryClient and initializes all the module query clients
func NewQueryClient(grpcEndpoint string) (*QueryClient, error) {
	conn, err := newGrpcConnection(context.Background(), grpcEndpoint)
	if err != nil {
		return &QueryClient{}, err
	}
	client := &QueryClient{
		Tm:           tmservice.NewServiceClient(conn),
		Tx:           txtypes.NewServiceClient(conn),
		Auth:         authtypes.NewQueryClient(conn),
		Authz:        authz.NewQueryClient(conn),
		Bank:         banktypes.NewQueryClient(conn),
		Distribution: disttypes.NewQueryClient(conn),
		Evidence:     evidencetypes.NewQueryClient(conn),
		Gov:          govv1types.NewQueryClient(conn),
		GovBeta:      govv1beta1types.NewQueryClient(conn),
		Mint:         minttypes.NewQueryClient(conn),
		Params:       paramstypes.NewQueryClient(conn),
		Slashing:     slashingtypes.NewQueryClient(conn),
		Staking:      stakingtypes.NewQueryClient(conn),
		Upgrade:      upgradetypes.NewQueryClient(conn),
		Consensus:    consensustypes.NewQueryClient(conn),

		Evm:         evmtypes.NewQueryClient(conn),
		Feemarket:   feemarkettypes.NewQueryClient(conn),
		IbcClient:   ibcclienttypes.NewQueryClient(conn),
		IbcChannel:  ibcchanneltypes.NewQueryClient(conn),
		IbcTransfer: ibctransfertypes.NewQueryClient(conn),

		Auction:   auctiontypes.NewQueryClient(conn),
		Bep3:      bep3types.NewQueryClient(conn),
		Cdp:       cdptypes.NewQueryClient(conn),
		Committee: committeetypes.NewQueryClient(conn),
		Community: communitytypes.NewQueryClient(conn),
		Earn:      earntypes.NewQueryClient(conn),
		Evmutil:   evmutiltypes.NewQueryClient(conn),
		Hard:      hardtypes.NewQueryClient(conn),
		Incentive: incentivetypes.NewQueryClient(conn),
		Issuance:  issuancetypes.NewQueryClient(conn),
		Istdist:   istdisttypes.NewQueryClient(conn),
		Liquid:    liquidtypes.NewQueryClient(conn),
		Pricefeed: pricefeedtypes.NewQueryClient(conn),
		Savings:   savingstypes.NewQueryClient(conn),
		Swap:      swaptypes.NewQueryClient(conn),
	}
	return client, nil
}
