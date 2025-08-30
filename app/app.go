package app

import (
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"path/filepath"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	consensus "github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/types"
	transfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclientclient "github.com/cosmos/ibc-go/v7/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcporttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	evmante "github.com/evmos/ethermint/app/ante"
	ethermintconfig "github.com/evmos/ethermint/server/config"
	"github.com/evmos/ethermint/x/evm"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/evmos/ethermint/x/evm/vm/geth"
	"github.com/evmos/ethermint/x/feemarket"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
	"github.com/gorilla/mux"

	"github.com/istchain/istchain/app/ante"
	"github.com/istchain/istchain/x/auction"
	auctionkeeper "github.com/istchain/istchain/x/auction/keeper"
	auctiontypes "github.com/istchain/istchain/x/auction/types"
	"github.com/istchain/istchain/x/bep3"
	bep3keeper "github.com/istchain/istchain/x/bep3/keeper"
	bep3types "github.com/istchain/istchain/x/bep3/types"
	"github.com/istchain/istchain/x/cdp"
	cdpkeeper "github.com/istchain/istchain/x/cdp/keeper"
	cdptypes "github.com/istchain/istchain/x/cdp/types"
	"github.com/istchain/istchain/x/committee"
	committeeclient "github.com/istchain/istchain/x/committee/client"
	committeekeeper "github.com/istchain/istchain/x/committee/keeper"
	committeetypes "github.com/istchain/istchain/x/committee/types"
	"github.com/istchain/istchain/x/community"
	communityclient "github.com/istchain/istchain/x/community/client"
	communitykeeper "github.com/istchain/istchain/x/community/keeper"
	communitytypes "github.com/istchain/istchain/x/community/types"
	earn "github.com/istchain/istchain/x/earn"
	earnclient "github.com/istchain/istchain/x/earn/client"
	earnkeeper "github.com/istchain/istchain/x/earn/keeper"
	earntypes "github.com/istchain/istchain/x/earn/types"
	evmutil "github.com/istchain/istchain/x/evmutil"
	evmutilkeeper "github.com/istchain/istchain/x/evmutil/keeper"
	evmutiltypes "github.com/istchain/istchain/x/evmutil/types"
	"github.com/istchain/istchain/x/hard"
	hardkeeper "github.com/istchain/istchain/x/hard/keeper"
	hardtypes "github.com/istchain/istchain/x/hard/types"
	"github.com/istchain/istchain/x/incentive"
	incentivekeeper "github.com/istchain/istchain/x/incentive/keeper"
	incentivetypes "github.com/istchain/istchain/x/incentive/types"
	issuance "github.com/istchain/istchain/x/issuance"
	issuancekeeper "github.com/istchain/istchain/x/issuance/keeper"
	issuancetypes "github.com/istchain/istchain/x/issuance/types"
	"github.com/istchain/istchain/x/istdist"
	istdistclient "github.com/istchain/istchain/x/istdist/client"
	istdistkeeper "github.com/istchain/istchain/x/istdist/keeper"
	istdisttypes "github.com/istchain/istchain/x/istdist/types"
	"github.com/istchain/istchain/x/liquid"
	liquidkeeper "github.com/istchain/istchain/x/liquid/keeper"
	liquidtypes "github.com/istchain/istchain/x/liquid/types"
	metrics "github.com/istchain/istchain/x/metrics"
	metricstypes "github.com/istchain/istchain/x/metrics/types"
	pricefeed "github.com/istchain/istchain/x/pricefeed"
	pricefeedkeeper "github.com/istchain/istchain/x/pricefeed/keeper"
	pricefeedtypes "github.com/istchain/istchain/x/pricefeed/types"
	"github.com/istchain/istchain/x/router"
	routerkeeper "github.com/istchain/istchain/x/router/keeper"
	routertypes "github.com/istchain/istchain/x/router/types"
	savings "github.com/istchain/istchain/x/savings"
	savingskeeper "github.com/istchain/istchain/x/savings/keeper"
	savingstypes "github.com/istchain/istchain/x/savings/types"
	"github.com/istchain/istchain/x/swap"
	swapkeeper "github.com/istchain/istchain/x/swap/keeper"
	swaptypes "github.com/istchain/istchain/x/swap/types"
	validatorvesting "github.com/istchain/istchain/x/validator-vesting"
	validatorvestingrest "github.com/istchain/istchain/x/validator-vesting/client/rest"
	validatorvestingtypes "github.com/istchain/istchain/x/validator-vesting/types"
)

const appName = "istchain"

var (
	// DefaultNodeHome default home directory for the application daemon
	DefaultNodeHome string

	// ModuleBasics manages simple versions of full app modules.
	ModuleBasics = module.NewBasicManager(
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic([]govclient.ProposalHandler{
			paramsclient.ProposalHandler,
			upgradeclient.LegacyProposalHandler,
			upgradeclient.LegacyCancelProposalHandler,
			ibcclientclient.UpdateClientProposalHandler,
			ibcclientclient.UpgradeProposalHandler,
			istdistclient.ProposalHandler,
			committeeclient.ProposalHandler,
			earnclient.DepositProposalHandler,
			earnclient.WithdrawProposalHandler,
			communityclient.LendDepositProposalHandler,
			communityclient.LendWithdrawProposalHandler,
		}),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		solomachine.AppModuleBasic{},
		packetforward.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		transfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		evm.AppModuleBasic{},
		feemarket.AppModuleBasic{},
		istdist.AppModuleBasic{},
		auction.AppModuleBasic{},
		issuance.AppModuleBasic{},
		bep3.AppModuleBasic{},
		pricefeed.AppModuleBasic{},
		swap.AppModuleBasic{},
		cdp.AppModuleBasic{},
		hard.AppModuleBasic{},
		committee.AppModuleBasic{},
		incentive.AppModuleBasic{},
		savings.AppModuleBasic{},
		validatorvesting.AppModuleBasic{},
		evmutil.AppModuleBasic{},
		liquid.AppModuleBasic{},
		earn.AppModuleBasic{},
		router.AppModuleBasic{},
		mint.AppModuleBasic{},
		community.AppModuleBasic{},
		metrics.AppModuleBasic{},
		consensus.AppModuleBasic{},
	)

	// module account permissions (unchanged)
	mAccPerms = map[string][]string{
		authtypes.FeeCollectorName:      nil,
		distrtypes.ModuleName:           nil,
		stakingtypes.BondedPoolName:     {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName:  {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:             {authtypes.Burner},
		ibctransfertypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
		evmtypes.ModuleName:             {authtypes.Minter, authtypes.Burner},
		evmutiltypes.ModuleName:         {authtypes.Minter, authtypes.Burner},
		istdisttypes.IstDistMacc:        {authtypes.Minter},
		auctiontypes.ModuleName:         nil,
		issuancetypes.ModuleAccountName: {authtypes.Minter, authtypes.Burner},
		bep3types.ModuleName:            {authtypes.Burner, authtypes.Minter},
		swaptypes.ModuleName:            nil,
		cdptypes.ModuleName:             {authtypes.Minter, authtypes.Burner},
		cdptypes.LiquidatorMacc:         {authtypes.Minter, authtypes.Burner},
		hardtypes.ModuleAccountName:     {authtypes.Minter},
		savingstypes.ModuleAccountName:  nil,
		liquidtypes.ModuleAccountName:   {authtypes.Minter, authtypes.Burner},
		earntypes.ModuleAccountName:     nil,
		istdisttypes.FundModuleAccount:  nil,
		minttypes.ModuleName:            {authtypes.Minter},
		communitytypes.ModuleName:       nil,
	}
)

// Verify app interface at compile time
var _ servertypes.Application = (*App)(nil)

// Options bundles several configuration params for an App.
type Options struct {
	SkipLoadLatest        bool
	SkipUpgradeHeights    map[int64]bool
	SkipGenesisInvariants bool
	InvariantCheckPeriod  uint
	MempoolEnableAuth     bool
	MempoolAuthAddresses  []sdk.AccAddress
	EVMTrace              string
	EVMMaxGasWanted       uint64
	TelemetryOptions      metricstypes.TelemetryOptions
}

// DefaultOptions is a sensible default Options value.
var DefaultOptions = Options{
	EVMTrace:        ethermintconfig.DefaultEVMTracer,
	EVMMaxGasWanted: ethermintconfig.DefaultMaxTxGasWanted,
}

// App is the IstChain ABCI application.
type App struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	// store keys
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	accountKeeper         authkeeper.AccountKeeper
	bankKeeper            bankkeeper.Keeper
	capabilityKeeper      *capabilitykeeper.Keeper
	stakingKeeper         *stakingkeeper.Keeper
	distrKeeper           distrkeeper.Keeper
	govKeeper             govkeeper.Keeper
	paramsKeeper          paramskeeper.Keeper
	authzKeeper           authzkeeper.Keeper
	crisisKeeper          crisiskeeper.Keeper
	slashingKeeper        slashingkeeper.Keeper
	ibcKeeper             *ibckeeper.Keeper
	packetForwardKeeper   *packetforwardkeeper.Keeper
	evmKeeper             *evmkeeper.Keeper
	evmutilKeeper         evmutilkeeper.Keeper
	feeMarketKeeper       feemarketkeeper.Keeper
	upgradeKeeper         upgradekeeper.Keeper
	evidenceKeeper        evidencekeeper.Keeper
	transferKeeper        ibctransferkeeper.Keeper
	istdistKeeper         istdistkeeper.Keeper
	auctionKeeper         auctionkeeper.Keeper
	issuanceKeeper        issuancekeeper.Keeper
	bep3Keeper            bep3keeper.Keeper
	pricefeedKeeper       pricefeedkeeper.Keeper
	swapKeeper            swapkeeper.Keeper
	cdpKeeper             cdpkeeper.Keeper
	hardKeeper            hardkeeper.Keeper
	committeeKeeper       committeekeeper.Keeper
	incentiveKeeper       incentivekeeper.Keeper
	savingsKeeper         savingskeeper.Keeper
	liquidKeeper          liquidkeeper.Keeper
	earnKeeper            earnkeeper.Keeper
	routerKeeper          routerkeeper.Keeper
	mintKeeper            mintkeeper.Keeper
	communityKeeper       communitykeeper.Keeper
	consensusParamsKeeper consensusparamkeeper.Keeper

	// scoped keepers
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// module manager & configurator
	mm           *module.Manager
	sm           *module.SimulationManager
	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil || userHomeDir == "" {
		// 回退到当前工作目录，避免空路径
		stdlog.Printf("failed to get home dir: %v", err)
		DefaultNodeHome = filepath.Join(".", ".istchain")
		return
	}
	DefaultNodeHome = filepath.Join(userHomeDir, ".istchain")
}

// BEGIN/END/INIT order slices (behavior unchanged; extracted for readability)
var beginBlockOrder = []string{
	metricstypes.ModuleName,
	upgradetypes.ModuleName,
	capabilitytypes.ModuleName,
	committeetypes.ModuleName,
	communitytypes.ModuleName,
	minttypes.ModuleName,
	distrtypes.ModuleName,
	slashingtypes.ModuleName,
	evidencetypes.ModuleName,
	stakingtypes.ModuleName,
	feemarkettypes.ModuleName,
	evmtypes.ModuleName,
	istdisttypes.ModuleName,
	auctiontypes.ModuleName,
	cdptypes.ModuleName,
	bep3types.ModuleName,
	hardtypes.ModuleName,
	issuancetypes.ModuleName,
	incentivetypes.ModuleName,
	ibcexported.ModuleName,
	// remaining modules with empty begin blocker
	swaptypes.ModuleName,
	vestingtypes.ModuleName,
	pricefeedtypes.ModuleName,
	validatorvestingtypes.ModuleName,
	authtypes.ModuleName,
	banktypes.ModuleName,
	govtypes.ModuleName,
	crisistypes.ModuleName,
	genutiltypes.ModuleName,
	ibctransfertypes.ModuleName,
	paramstypes.ModuleName,
	authz.ModuleName,
	evmutiltypes.ModuleName,
	savingstypes.ModuleName,
	liquidtypes.ModuleName,
	earntypes.ModuleName,
	routertypes.ModuleName,
	consensusparamtypes.ModuleName,
	packetforwardtypes.ModuleName,
}

var endBlockOrder = []string{
	crisistypes.ModuleName,
	govtypes.ModuleName,
	stakingtypes.ModuleName,
	evmtypes.ModuleName,
	feemarkettypes.ModuleName,
	pricefeedtypes.ModuleName,
	// remaining modules with empty end blocker
	capabilitytypes.ModuleName,
	incentivetypes.ModuleName,
	issuancetypes.ModuleName,
	slashingtypes.ModuleName,
	distrtypes.ModuleName,
	auctiontypes.ModuleName,
	bep3types.ModuleName,
	cdptypes.ModuleName,
	hardtypes.ModuleName,
	committeetypes.ModuleName,
	upgradetypes.ModuleName,
	evidencetypes.ModuleName,
	istdisttypes.ModuleName,
	swaptypes.ModuleName,
	vestingtypes.ModuleName,
	ibcexported.ModuleName,
	validatorvestingtypes.ModuleName,
	authtypes.ModuleName,
	banktypes.ModuleName,
	genutiltypes.ModuleName,
	ibctransfertypes.ModuleName,
	paramstypes.ModuleName,
	authz.ModuleName,
	evmutiltypes.ModuleName,
	savingstypes.ModuleName,
	liquidtypes.ModuleName,
	earntypes.ModuleName,
	routertypes.ModuleName,
	minttypes.ModuleName,
	communitytypes.ModuleName,
	metricstypes.ModuleName,
	consensusparamtypes.ModuleName,
	packetforwardtypes.ModuleName,
}

var initGenesisOrder = []string{
	capabilitytypes.ModuleName,
	authtypes.ModuleName,
	banktypes.ModuleName,
	distrtypes.ModuleName,
	stakingtypes.ModuleName,
	slashingtypes.ModuleName,
	govtypes.ModuleName,
	minttypes.ModuleName,
	ibcexported.ModuleName,
	evidencetypes.ModuleName,
	authz.ModuleName,
	ibctransfertypes.ModuleName,
	evmtypes.ModuleName,
	feemarkettypes.ModuleName,
	istdisttypes.ModuleName,
	auctiontypes.ModuleName,
	issuancetypes.ModuleName,
	savingstypes.ModuleName,
	bep3types.ModuleName,
	pricefeedtypes.ModuleName,
	swaptypes.ModuleName,
	cdptypes.ModuleName,
	hardtypes.ModuleName,
	incentivetypes.ModuleName,
	committeetypes.ModuleName,
	evmutiltypes.ModuleName,
	earntypes.ModuleName,
	communitytypes.ModuleName,
	genutiltypes.ModuleName,
	// remaining modules with empty InitGenesis
	vestingtypes.ModuleName,
	paramstypes.ModuleName,
	upgradetypes.ModuleName,
	validatorvestingtypes.ModuleName,
	liquidtypes.ModuleName,
	routertypes.ModuleName,
	metricstypes.ModuleName,
	consensusparamtypes.ModuleName,
	packetforwardtypes.ModuleName,
	crisistypes.ModuleName,
}

// NewApp returns a reference to an initialized App.
func NewApp(
	logger tmlog.Logger,
	db dbm.DB,
	homePath string,
	traceStore io.Writer,
	encodingConfig istparams.EncodingConfig, // assumed available in project
	options Options,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	appCodec := encodingConfig.Marshaler
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	bApp := baseapp.NewBaseApp(appName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		distrtypes.StoreKey, slashingtypes.StoreKey, packetforwardtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey,
		upgradetypes.StoreKey, evidencetypes.StoreKey, ibctransfertypes.StoreKey,
		evmtypes.StoreKey, feemarkettypes.StoreKey, authzkeeper.StoreKey,
		capabilitytypes.StoreKey, istdisttypes.StoreKey, auctiontypes.StoreKey,
		issuancetypes.StoreKey, bep3types.StoreKey, pricefeedtypes.StoreKey,
		swaptypes.StoreKey, cdptypes.StoreKey, hardtypes.StoreKey, communitytypes.StoreKey,
		committeetypes.StoreKey, incentivetypes.StoreKey, evmutiltypes.StoreKey,
		savingstypes.StoreKey, earntypes.StoreKey, minttypes.StoreKey,
		consensusparamtypes.StoreKey, crisistypes.StoreKey,
	)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey, evmtypes.TransientKey, feemarkettypes.TransientKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	// Authority for gov proposals
	govAuthAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	govAuthAddrStr := govAuthAddr.String()

	app := &App{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// params keeper & subspaces
	app.paramsKeeper = paramskeeper.NewKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	authSubspace := app.paramsKeeper.Subspace(authtypes.ModuleName)
	bankSubspace := app.paramsKeeper.Subspace(banktypes.ModuleName)
	stakingSubspace := app.paramsKeeper.Subspace(stakingtypes.ModuleName)
	distrSubspace := app.paramsKeeper.Subspace(distrtypes.ModuleName)
	slashingSubspace := app.paramsKeeper.Subspace(slashingtypes.ModuleName)
	govSubspace := app.paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	crisisSubspace := app.paramsKeeper.Subspace(crisistypes.ModuleName)
	istdistSubspace := app.paramsKeeper.Subspace(istdisttypes.ModuleName)
	auctionSubspace := app.paramsKeeper.Subspace(auctiontypes.ModuleName)
	issuanceSubspace := app.paramsKeeper.Subspace(issuancetypes.ModuleName)
	bep3Subspace := app.paramsKeeper.Subspace(bep3types.ModuleName)
	pricefeedSubspace := app.paramsKeeper.Subspace(pricefeedtypes.ModuleName)
	swapSubspace := app.paramsKeeper.Subspace(swaptypes.ModuleName)
	cdpSubspace := app.paramsKeeper.Subspace(cdptypes.ModuleName)
	hardSubspace := app.paramsKeeper.Subspace(hardtypes.ModuleName)
	incentiveSubspace := app.paramsKeeper.Subspace(incentivetypes.ModuleName)
	savingsSubspace := app.paramsKeeper.Subspace(savingstypes.ModuleName)
	ibcSubspace := app.paramsKeeper.Subspace(ibcexported.ModuleName)
	ibctransferSubspace := app.paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	packetforwardSubspace := app.paramsKeeper.Subspace(packetforwardtypes.ModuleName)
	feemarketSubspace := app.paramsKeeper.Subspace(feemarkettypes.ModuleName)
	evmSubspace := app.paramsKeeper.Subspace(evmtypes.ModuleName)
	evmutilSubspace := app.paramsKeeper.Subspace(evmutiltypes.ModuleName)
	earnSubspace := app.paramsKeeper.Subspace(earntypes.ModuleName)
	mintSubspace := app.paramsKeeper.Subspace(minttypes.ModuleName)

	// consensus params keeper
	app.consensusParamsKeeper = consensusparamkeeper.NewKeeper(appCodec, keys[consensusparamtypes.StoreKey], govAuthAddrStr)
	bApp.SetParamStore(&app.consensusParamsKeeper)

	// capability keeper & scopes
	app.capabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	scopedIBCKeeper := app.capabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedTransferKeeper := app.capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	app.capabilityKeeper.Seal()

	// core keepers
	app.accountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], authtypes.ProtoBaseAccount, mAccPerms,
		sdk.GetConfig().GetBech32AccountAddrPrefix(), govAuthAddrStr,
	)
	app.bankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], app.accountKeeper, app.loadBlockedMaccAddrs(), govAuthAddrStr,
	)
	app.stakingKeeper = stakingkeeper.NewKeeper(appCodec, keys[stakingtypes.StoreKey], app.accountKeeper, app.bankKeeper, govAuthAddrStr)
	app.authzKeeper = authzkeeper.NewKeeper(keys[authzkeeper.StoreKey], appCodec, app.BaseApp.MsgServiceRouter(), app.accountKeeper)
	app.distrKeeper = distrkeeper.NewKeeper(appCodec, keys[distrtypes.StoreKey], app.accountKeeper, app.bankKeeper, app.stakingKeeper, authtypes.FeeCollectorName, govAuthAddrStr)
	app.slashingKeeper = slashingkeeper.NewKeeper(appCodec, app.legacyAmino, keys[slashingtypes.StoreKey], app.stakingKeeper, govAuthAddrStr)
	app.crisisKeeper = *crisiskeeper.NewKeeper(appCodec, keys[crisistypes.StoreKey], options.InvariantCheckPeriod, app.bankKeeper, authtypes.FeeCollectorName, govAuthAddrStr)
	app.upgradeKeeper = *upgradekeeper.NewKeeper(options.SkipUpgradeHeights, keys[upgradetypes.StoreKey], appCodec, homePath, app.BaseApp, govAuthAddrStr)
	app.evidenceKeeper = *evidencekeeper.NewKeeper(appCodec, keys[evidencetypes.StoreKey], app.stakingKeeper, app.slashingKeeper)

	app.ibcKeeper = ibckeeper.NewKeeper(appCodec, keys[ibcexported.StoreKey], ibcSubspace, app.stakingKeeper, app.upgradeKeeper, scopedIBCKeeper)

	// Ethermint keepers
	app.feeMarketKeeper = feemarketkeeper.NewKeeper(appCodec, govAuthAddr, keys[feemarkettypes.StoreKey], tkeys[feemarkettypes.TransientKey], feemarketSubspace)
	app.evmutilKeeper = evmutilkeeper.NewKeeper(app.appCodec, keys[evmutiltypes.StoreKey], evmutilSubspace, app.bankKeeper, app.accountKeeper)
	evmBankKeeper := evmutilkeeper.NewEvmBankKeeper(app.evmutilKeeper, app.bankKeeper, app.accountKeeper)
	app.evmKeeper = evmkeeper.NewKeeper(
		appCodec, keys[evmtypes.StoreKey], tkeys[evmtypes.TransientKey], govAuthAddr,
		app.accountKeeper, evmBankKeeper, app.stakingKeeper, app.feeMarketKeeper,
		nil, // precompiled contracts
		geth.NewEVM, options.EVMTrace, evmSubspace,
	)
	app.evmutilKeeper.SetEvmKeeper(app.evmKeeper)

	// PFM must be initialized before Transfer
	app.packetForwardKeeper = packetforwardkeeper.NewKeeper(
		appCodec, keys[packetforwardtypes.StoreKey], nil,
		app.ibcKeeper.ChannelKeeper, app.bankKeeper, app.ibcKeeper.ChannelKeeper, govAuthAddrStr,
	)
	app.transferKeeper = ibctransferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey], ibctransferSubspace, app.packetForwardKeeper,
		app.ibcKeeper.ChannelKeeper, &app.ibcKeeper.PortKeeper, app.accountKeeper, app.bankKeeper, scopedTransferKeeper,
	)
	app.packetForwardKeeper.SetTransferKeeper(app.transferKeeper)
	transferModule := transfer.NewAppModule(app.transferKeeper)

	// ibc router
	var transferStack ibcporttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.transferKeeper)
	transferStack = packetforward.NewIBCMiddleware(
		transferStack, app.packetForwardKeeper, 0,
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
		packetforwardkeeper.DefaultRefundTransferPacketTimeoutTimestamp,
	)
	ibcRouter := ibcporttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
	app.ibcKeeper.SetRouter(ibcRouter)

	// istchain custom keepers
	app.auctionKeeper = auctionkeeper.NewKeeper(appCodec, keys[auctiontypes.StoreKey], auctionSubspace, app.bankKeeper, app.accountKeeper)
	app.issuanceKeeper = issuancekeeper.NewKeeper(appCodec, keys[issuancetypes.StoreKey], issuanceSubspace, app.accountKeeper, app.bankKeeper)
	app.bep3Keeper = bep3keeper.NewKeeper(appCodec, keys[bep3types.StoreKey], app.bankKeeper, app.accountKeeper, bep3Subspace, app.ModuleAccountAddrs())
	app.pricefeedKeeper = pricefeedkeeper.NewKeeper(appCodec, keys[pricefeedtypes.StoreKey], pricefeedSubspace)
	swapKeeper := swapkeeper.NewKeeper(appCodec, keys[swaptypes.StoreKey], swapSubspace, app.accountKeeper, app.bankKeeper)
	cdpKeeper := cdpkeeper.NewKeeper(appCodec, keys[cdptypes.StoreKey], cdpSubspace, app.pricefeedKeeper, app.auctionKeeper, app.bankKeeper, app.accountKeeper, mAccPerms)
	hardKeeper := hardkeeper.NewKeeper(appCodec, keys[hardtypes.StoreKey], hardSubspace, app.accountKeeper, app.bankKeeper, app.pricefeedKeeper, app.auctionKeeper)
	app.liquidKeeper = liquidkeeper.NewDefaultKeeper(appCodec, app.accountKeeper, app.bankKeeper, app.stakingKeeper, &app.distrKeeper)
	savingsKeeper := savingskeeper.NewKeeper(appCodec, keys[savingstypes.StoreKey], savingsSubspace, app.accountKeeper, app.bankKeeper, app.liquidKeeper)
	earnKeeper := earnkeeper.NewKeeper(appCodec, keys[earntypes.StoreKey], earnSubspace, app.accountKeeper, app.bankKeeper, &app.liquidKeeper, &hardKeeper, &savingsKeeper, &app.distrKeeper)

	app.istdistKeeper = istdistkeeper.NewKeeper(appCodec, keys[istdisttypes.StoreKey], istdistSubspace, app.bankKeeper, app.accountKeeper, app.distrKeeper, app.loadBlockedMaccAddrs())
	app.mintKeeper = mintkeeper.NewKeeper(appCodec, keys[minttypes.StoreKey], app.stakingKeeper, app.accountKeeper, app.bankKeeper, authtypes.FeeCollectorName, govAuthAddrStr)

	app.communityKeeper = communitykeeper.NewKeeper(
		appCodec, keys[communitytypes.StoreKey], app.accountKeeper, app.bankKeeper,
		&cdpKeeper, app.distrKeeper, &hardKeeper, &app.mintKeeper, &app.istdistKeeper, app.stakingKeeper, govAuthAddr,
	)

	app.incentiveKeeper = incentivekeeper.NewKeeper(
		appCodec, keys[incentivetypes.StoreKey], incentiveSubspace, app.bankKeeper,
		&cdpKeeper, &hardKeeper, app.accountKeeper, app.stakingKeeper,
		&swapKeeper, &savingsKeeper, &app.liquidKeeper, &earnKeeper,
		app.mintKeeper, app.distrKeeper, app.pricefeedKeeper,
	)
	app.routerKeeper = routerkeeper.NewKeeper(&app.earnKeeper, app.liquidKeeper, app.stakingKeeper)

	// hooks
	app.stakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(app.distrKeeper.Hooks(), app.slashingKeeper.Hooks(), app.incentiveKeeper.Hooks()))
	app.swapKeeper = *swapKeeper.SetHooks(app.incentiveKeeper.Hooks())
	app.cdpKeeper = *cdpKeeper.SetHooks(cdptypes.NewMultiCDPHooks(app.incentiveKeeper.Hooks()))
	app.hardKeeper = *hardKeeper.SetHooks(hardtypes.NewMultiHARDHooks(app.incentiveKeeper.Hooks()))
	app.savingsKeeper = savingsKeeper
	app.earnKeeper = *earnKeeper.SetHooks(app.incentiveKeeper.Hooks())

	// committee router & keeper
	committeeGovRouter := govv1beta1.NewRouter().
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(communitytypes.RouterKey, community.NewCommunityPoolProposalHandler(app.communityKeeper)).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(&app.upgradeKeeper))
	app.committeeKeeper = committeekeeper.NewKeeper(
		appCodec, keys[committeetypes.StoreKey], committeeGovRouter,
		app.paramsKeeper, app.accountKeeper, app.bankKeeper,
	)

	// gov router & keeper
	govRouter := govv1beta1.NewRouter().
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(&app.upgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.ibcKeeper.ClientKeeper)).
		AddRoute(istdisttypes.RouterKey, istdist.NewCommunityPoolMultiSpendProposalHandler(app.istdistKeeper)).
		AddRoute(earntypes.RouterKey, earn.NewCommunityPoolProposalHandler(app.earnKeeper)).
		AddRoute(communitytypes.RouterKey, community.NewCommunityPoolProposalHandler(app.communityKeeper)).
		AddRoute(committeetypes.RouterKey, committee.NewProposalHandler(app.committeeKeeper))

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(appCodec, keys[govtypes.StoreKey], app.accountKeeper, app.bankKeeper, app.stakingKeeper, app.MsgServiceRouter(), govConfig, govAuthAddrStr)
	govKeeper.SetLegacyRouter(govRouter)
	app.govKeeper = *govKeeper

	// override tally handler
	tallyHandler := NewTallyHandler(app.govKeeper, *app.stakingKeeper, app.savingsKeeper, app.earnKeeper, app.liquidKeeper, app.bankKeeper)
	app.govKeeper.SetTallyHandler(tallyHandler)

	// module manager
	app.mm = module.NewManager(
		genutil.NewAppModule(app.accountKeeper, app.stakingKeeper, app.BaseApp.DeliverTx, encodingConfig.TxConfig),
		auth.NewAppModule(appCodec, app.accountKeeper, authsims.RandomGenesisAccounts, authSubspace),
		bank.NewAppModule(appCodec, app.bankKeeper, app.accountKeeper, bankSubspace),
		capability.NewAppModule(appCodec, *app.capabilityKeeper, false),
		staking.NewAppModule(appCodec, app.stakingKeeper, app.accountKeeper, app.bankKeeper, stakingSubspace),
		distr.NewAppModule(appCodec, app.distrKeeper, app.accountKeeper, app.bankKeeper, app.stakingKeeper, distrSubspace),
		gov.NewAppModule(appCodec, &app.govKeeper, app.accountKeeper, app.bankKeeper, govSubspace),
		params.NewAppModule(app.paramsKeeper),
		crisis.NewAppModule(&app.crisisKeeper, options.SkipGenesisInvariants, crisisSubspace),
		slashing.NewAppModule(appCodec, app.slashingKeeper, app.accountKeeper, app.bankKeeper, app.stakingKeeper, slashingSubspace),
		consensus.NewAppModule(appCodec, app.consensusParamsKeeper),
		ibc.NewAppModule(app.ibcKeeper),
		packetforward.NewAppModule(app.packetForwardKeeper, packetforwardSubspace),
		evm.NewAppModule(app.evmKeeper, app.accountKeeper),
		feemarket.NewAppModule(app.feeMarketKeeper, feemarketSubspace),
		upgrade.NewAppModule(&app.upgradeKeeper),
		evidence.NewAppModule(app.evidenceKeeper),
		transferModule,
		vesting.NewAppModule(app.accountKeeper, app.bankKeeper),
		authzmodule.NewAppModule(appCodec, app.authzKeeper, app.accountKeeper, app.bankKeeper, app.interfaceRegistry),
		istdist.NewAppModule(app.istdistKeeper, app.accountKeeper),
		auction.NewAppModule(app.auctionKeeper, app.accountKeeper, app.bankKeeper),
		issuance.NewAppModule(app.issuanceKeeper, app.accountKeeper, app.bankKeeper),
		bep3.NewAppModule(app.bep3Keeper, app.accountKeeper, app.bankKeeper),
		pricefeed.NewAppModule(app.pricefeedKeeper, app.accountKeeper),
		validatorvesting.NewAppModule(app.bankKeeper),
		swap.NewAppModule(app.swapKeeper, app.accountKeeper),
		cdp.NewAppModule(app.cdpKeeper, app.accountKeeper, app.pricefeedKeeper, app.bankKeeper),
		hard.NewAppModule(app.hardKeeper, app.accountKeeper, app.bankKeeper, app.pricefeedKeeper),
		committee.NewAppModule(app.committeeKeeper, app.accountKeeper),
		incentive.NewAppModule(app.incentiveKeeper, app.accountKeeper, app.bankKeeper, app.cdpKeeper),
		evmutil.NewAppModule(app.evmutilKeeper, app.bankKeeper, app.accountKeeper),
		savings.NewAppModule(app.savingsKeeper, app.accountKeeper, app.bankKeeper),
		liquid.NewAppModule(app.liquidKeeper),
		earn.NewAppModule(app.earnKeeper, app.accountKeeper, app.bankKeeper),
		router.NewAppModule(app.routerKeeper),
		mint.NewAppModule(appCodec, app.mintKeeper, app.accountKeeper, nil, mintSubspace),
		community.NewAppModule(app.communityKeeper, app.accountKeeper),
		metrics.NewAppModule(options.TelemetryOptions),
	)

	// orderings (extracted; unchanged)
	app.mm.SetOrderBeginBlockers(beginBlockOrder...)
	app.mm.SetOrderEndBlockers(endBlockOrder...)
	app.mm.SetOrderInitGenesis(initGenesisOrder...)

	app.mm.RegisterInvariants(&app.crisisKeeper)

	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.RegisterServices(app.configurator)
	app.RegisterUpgradeHandlers()

	// stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// ante handler
	var fetchers []ante.AddressFetcher
	if options.MempoolEnableAuth {
		fetchers = append(fetchers,
			func(sdk.Context) []sdk.AccAddress { return options.MempoolAuthAddresses },
			app.bep3Keeper.GetAuthorizedAddresses,
			app.pricefeedKeeper.GetAuthorizedAddresses,
		)
	}
	anteOptions := ante.HandlerOptions{
		AccountKeeper:          app.accountKeeper,
		BankKeeper:             app.bankKeeper,
		EvmKeeper:              app.evmKeeper,
		IBCKeeper:              app.ibcKeeper,
		FeeMarketKeeper:        app.feeMarketKeeper,
		SignModeHandler:        encodingConfig.TxConfig.SignModeHandler(),
		SigGasConsumer:         evmante.DefaultSigVerificationGasConsumer,
		MaxTxGasWanted:         options.EVMMaxGasWanted,
		AddressFetchers:        fetchers,
		ExtensionOptionChecker: nil,
		TxFeeChecker:           nil,
	}
	antehandler, err := ante.NewAnteHandler(anteOptions)
	if err != nil {
		panic(fmt.Errorf("failed to create antehandler: %w", err))
	}
	app.SetAnteHandler(antehandler)

	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// load latest
	if !options.SkipLoadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(fmt.Errorf("failed to load latest version: %w", err))
		}
	}

	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper

	return app
}

func (app *App) RegisterServices(cfg module.Configurator) {
	app.mm.RegisterServices(cfg)
}

// BeginBlocker contains app specific logic for the BeginBlock abci call.
func (app *App) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker contains app specific logic for the EndBlock abci call.
func (app *App) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer contains app specific logic for the InitChain abci call.
func (app *App) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	// Store current module versions to enable in-place upgrades later.
	app.upgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads the app state for a particular height.
func (app *App) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *App) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range mAccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}
	return modAccAddrs
}

// InterfaceRegistry returns the app's InterfaceRegistry.
func (app *App) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// SimulationManager implements the SimulationApp interface.
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// custom REST routes
	validatorvestingrest.RegisterRoutes(clientCtx, apiSvr.Router)

	// rewrites must be registered before gateway routes (first-match wins)
	RegisterAPIRouteRewrites(apiSvr.Router)

	// GRPC Gateway routes
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	_ = apiConfig // swagger config intentionally unused (same as before)
}

// RegisterAPIRouteRewrites registers overwritten API routes that are
// registered after this function is called. Must be called before other route registrations.
func RegisterAPIRouteRewrites(router *mux.Router) {
	routeMap := map[string]string{
		// e.g. /cosmos/distribution/v1beta1/community_pool -> /istchain/community/v1beta1/total_balance
		"/cosmos/distribution/v1beta1/community_pool": "/istchain/community/v1beta1/total_balance",
	}
	for clientPath, backendPath := range routeMap {
		target := backendPath
		router.HandleFunc(clientPath, func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = target
			router.ServeHTTP(w, r)
		}).Methods("GET")
	}
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *App) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *App) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(clientCtx, app.BaseApp.GRPCQueryRouter(), app.interfaceRegistry, app.Query)
}

func (app *App) RegisterNodeService(clientCtx client.Context) {
	nodeservice.RegisterNodeService(clientCtx, app.BaseApp.GRPCQueryRouter())
}

// loadBlockedMaccAddrs returns a map indicating the blocked status of each module account address.
func (app *App) loadBlockedMaccAddrs() map[string]bool {
	modAccAddrs := app.ModuleAccountAddrs()

	allowed := map[string]bool{
		app.accountKeeper.GetModuleAddress(istdisttypes.ModuleName).String():          true, // istdist
		app.accountKeeper.GetModuleAddress(earntypes.ModuleName).String():             true, // earn
		app.accountKeeper.GetModuleAddress(liquidtypes.ModuleName).String():           true, // liquid
		app.accountKeeper.GetModuleAddress(istdisttypes.FundModuleAccount).String():   true, // istdist fund
		app.accountKeeper.GetModuleAddress(communitytypes.ModuleAccountName).String(): true, // community
		// NOTE: if adding evmutil, adjust the cosmos-coins-fully-backed-invariant accordingly.
	}
	for addr := range modAccAddrs {
		if allowed[addr] {
			modAccAddrs[addr] = false // unblocked
		}
	}
	return modAccAddrs
}

// GetMaccPerms returns a copy of the application's module account permissions.
func GetMaccPerms() map[string][]string {
	perms := make(map[string][]string, len(mAccPerms))
	for k, v := range mAccPerms {
		perms[k] = v
	}
	return perms
}
