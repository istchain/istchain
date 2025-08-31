package ante_test

import (
	"math/big"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/ethereum/eip712"
	"github.com/evmos/ethermint/tests"
	etherminttypes "github.com/evmos/ethermint/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
	"github.com/stretchr/testify/suite"

	"github.com/istchain/istchain/app"
	cdptypes "github.com/istchain/istchain/x/cdp/types"
	evmutilkeeper "github.com/istchain/istchain/x/evmutil/keeper"
	evmutiltestutil "github.com/istchain/istchain/x/evmutil/testutil"
	evmutiltypes "github.com/istchain/istchain/x/evmutil/types"
	hardtypes "github.com/istchain/istchain/x/hard/types"
	pricefeedtypes "github.com/istchain/istchain/x/pricefeed/types"
)

const (
	ChainID       = app.TestChainId
	USDCCoinDenom = "erc20/usdc"
	USDCCDPType   = "erc20-usdc"
	TxGas         = uint64(sims.DefaultGenTxGas * 10)

	// gas / denom constants
	gasDenom    = "uist"
	stableDenom = cdptypes.DefaultStableDenom

	// domain constants used by positive EIP-712 case
	domainName    = "IstChain Cosmos"
	domainVersion = "1.0.0"
)

// 10^18 (wei factor) — 使用常量替代运行时幂运算，避免中间转换及溢出风险。
var weiFactor = sdkmath.NewInt(1_000_000_000_000_000_000)

type EIP712TestSuite struct {
	suite.Suite

	tApp          app.TestApp
	ctx           sdk.Context
	evmutilKeeper evmutilkeeper.Keeper
	clientCtx     client.Context
	ethSigner     ethtypes.Signer
	testAddr      sdk.AccAddress
	testAddr2     sdk.AccAddress
	testPrivKey   cryptotypes.PrivKey
	testPrivKey2  cryptotypes.PrivKey
	testEVMAddr   evmutiltypes.InternalEVMAddress
	testEVMAddr2  evmutiltypes.InternalEVMAddress
	usdcEVMAddr   evmutiltypes.InternalEVMAddress
}

// ---- helpers ----------------------------------------------------------------

func (suite *EIP712TestSuite) getEVMAmount(amount int64) sdkmath.Int {
	return sdkmath.NewInt(amount).Mul(weiFactor)
}

func (suite *EIP712TestSuite) gasCoins(amt int64) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin(gasDenom, sdkmath.NewInt(amt)))
}

func (suite *EIP712TestSuite) mustEncodeTx(txBuilder client.TxBuilder) []byte {
	encodingConfig := app.MakeEncodingConfig()
	bz, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	suite.Require().NoError(err)
	return bz
}

func (suite *EIP712TestSuite) buildTransaction(
	gasAmount sdk.Coins,
	msgs []sdk.Msg,
	nonce uint64,
	pubKey cryptotypes.PubKey,
	option *codectypes.Any,
) client.TxBuilder {
	txBuilder := suite.clientCtx.TxConfig.NewTxBuilder()
	builder, ok := txBuilder.(authtx.ExtensionOptionsTxBuilder)
	suite.Require().True(ok)

	builder.SetExtensionOptions(option)
	builder.SetFeeAmount(gasAmount)
	builder.SetGasLimit(TxGas)

	sigsV2 := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			// EIP-712 走 amino-json 模式（ethermint 兼容路径）
			SignMode: signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
		},
		Sequence: nonce,
	}
	suite.Require().NoError(builder.SetSignatures(sigsV2))
	suite.Require().NoError(builder.SetMsgs(msgs...))
	return builder
}

func (suite *EIP712TestSuite) buildSigHash(
	from sdk.AccAddress,
	chainID string,
	gasAmount sdk.Coins,
	msgs []sdk.Msg,
	nonce uint64,
	customDomain *apitypes.TypedDataDomain,
	customDomainTypes []apitypes.Type,
) []byte {
	pc, err := etherminttypes.ParseChainID(chainID)
	suite.Require().NoError(err)
	ethChainID := pc.Uint64()

	// legacytx 仍用于 EIP-712 fee 表达
	fee := legacytx.NewStdFee(TxGas, gasAmount) //nolint:staticcheck
	accNumber := suite.tApp.GetAccountKeeper().GetAccount(suite.ctx, from).GetAccountNumber()

	data := eip712.ConstructUntypedEIP712Data(chainID, accNumber, nonce, 0, fee, msgs, "", nil)
	typedData, err := eip712.WrapTxToTypedData(
		ethChainID,
		msgs,
		data,
		&eip712.FeeDelegationOptions{FeePayer: from},
		suite.tApp.GetEvmKeeper().GetParams(suite.ctx),
	)
	suite.Require().NoError(err)

	if customDomain != nil {
		typedData.Domain = *customDomain
	}
	if customDomainTypes != nil {
		typedData.Types["EIP712Domain"] = customDomainTypes
	}

	sigHash, err := eip712.ComputeTypedDataHash(typedData)
	suite.Require().NoError(err)
	return sigHash
}

func (suite *EIP712TestSuite) createTestEIP712CosmosTxBuilderWithDomain(
	from sdk.AccAddress,
	priv cryptotypes.PrivKey,
	chainID string,
	gasAmount sdk.Coins,
	msgs []sdk.Msg,
	customDomain *apitypes.TypedDataDomain,
	customDomainTypes []apitypes.Type,
) client.TxBuilder {
	nonce, err := suite.tApp.GetAccountKeeper().GetSequence(suite.ctx, from)
	suite.Require().NoError(err)

	pc, err := etherminttypes.ParseChainID(chainID)
	suite.Require().NoError(err)
	ethChainID := pc.Uint64()

	sigHash := suite.buildSigHash(from, chainID, gasAmount, msgs, nonce, customDomain, customDomainTypes)

	keyringSigner := tests.NewSigner(priv)
	signature, pubKey, err := keyringSigner.SignByAddress(from, sigHash)
	suite.Require().NoError(err)
	// V: 0/1 -> 27/28
	signature[crypto.RecoveryIDOffset] += 27

	option, err := codectypes.NewAnyWithValue(&etherminttypes.ExtensionOptionsWeb3Tx{
		FeePayer:         from.String(),
		TypedDataChainID: ethChainID,
		FeePayerSig:      signature,
	})
	suite.Require().NoError(err)

	return suite.buildTransaction(gasAmount, msgs, nonce, pubKey, option)
}

func (suite *EIP712TestSuite) createTestEIP712CosmosTxBuilder(
	from sdk.AccAddress,
	priv cryptotypes.PrivKey,
	chainID string,
	gasAmount sdk.Coins,
	msgs []sdk.Msg,
) client.TxBuilder {
	return suite.createTestEIP712CosmosTxBuilderWithDomain(from, priv, chainID, gasAmount, msgs, nil, nil)
}

// 生成“存→铸→存款”消息流水（多数用例共享）
func (suite *EIP712TestSuite) depositMintLendMsgs(usdcDepositAmt int64, usdxToMintAmt int64) []sdk.Msg {
	usdcAmt := suite.getEVMAmount(usdcDepositAmt)
	convertMsg := evmutiltypes.NewMsgConvertERC20ToCoin(
		suite.testEVMAddr,
		suite.testAddr,
		suite.usdcEVMAddr,
		usdcAmt,
	)
	usdxAmt := sdkmath.NewInt(1_000_000).Mul(sdkmath.NewInt(usdxToMintAmt))
	mintMsg := cdptypes.NewMsgCreateCDP(
		suite.testAddr,
		sdk.NewCoin(USDCCoinDenom, usdcAmt),
		sdk.NewCoin(stableDenom, usdxAmt),
		USDCCDPType,
	)
	lendMsg := hardtypes.NewMsgDeposit(suite.testAddr, sdk.NewCoins(sdk.NewCoin(stableDenom, usdxAmt)))
	return []sdk.Msg{&convertMsg, &mintMsg, &lendMsg}
}

// 域与类型（正向用例使用）
func domainExpected(ethChainID int64) (apitypes.TypedDataDomain, []apitypes.Type) {
	return apitypes.TypedDataDomain{
			Name:              domainName,
			Version:           domainVersion,
			ChainId:           math.NewHexOrDecimal256(ethChainID),
			Salt:              "",
			VerifyingContract: "",
		}, []apitypes.Type{
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
		}
}

// ---- suite lifecycle --------------------------------------------------------

func (suite *EIP712TestSuite) SetupTest() {
	tApp := app.NewTestApp()
	suite.tApp = tApp
	suite.evmutilKeeper = tApp.GetEvmutilKeeper()

	// accounts
	addr, privkey := tests.NewAddrKey()
	suite.testAddr = sdk.AccAddress(addr.Bytes())
	suite.testPrivKey = privkey
	suite.testEVMAddr = evmutiltestutil.MustNewInternalEVMAddressFromString(addr.String())

	addr2, privKey2 := tests.NewAddrKey()
	suite.testAddr2 = sdk.AccAddress(addr2.Bytes())
	suite.testPrivKey2 = privKey2
	suite.testEVMAddr2 = evmutiltestutil.MustNewInternalEVMAddressFromString(addr2.String())

	encodingConfig := app.MakeEncodingConfig()
	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
	suite.ethSigner = ethtypes.LatestSignerForChainID(tApp.GetEvmKeeper().ChainID())

	// ---- genesis states
	cdc := tApp.AppCodec()

	evmGs := evmtypes.NewGenesisState(
		evmtypes.NewParams(
			"aist",
			false, // allowedUnprotectedTxs
			true,  // enableCreate
			true,  // enableCall
			evmtypes.DefaultChainConfig(),
			nil, // extraEIPs
			nil, // eip712AllowedMsgs
		),
		nil,
	)

	feemarketGenesis := feemarkettypes.DefaultGenesisState()
	feemarketGenesis.Params.EnableHeight = 1
	feemarketGenesis.Params.NoBaseFee = false

	cdpGenState := cdptypes.DefaultGenesisState()
	cdpGenState.Params.GlobalDebtLimit = sdk.NewInt64Coin("usdx", 53_000_000_000_000)
	cdpGenState.Params.CollateralParams = cdptypes.CollateralParams{
		{
			Denom:                            USDCCoinDenom,
			Type:                             USDCCDPType,
			LiquidationRatio:                 sdk.MustNewDecFromStr("1.01"),
			DebtLimit:                        sdk.NewInt64Coin("usdx", 500_000_000_000),
			StabilityFee:                     sdk.OneDec(),
			AuctionSize:                      sdkmath.NewIntFromUint64(10_000_000_000),
			LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
			CheckCollateralizationIndexCount: sdkmath.NewInt(10),
			KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
			SpotMarketID:                     "usdc:usd",
			LiquidationMarketID:              "usdc:usd:30",
			ConversionFactor:                 sdkmath.NewInt(18),
		},
	}

	hardGenState := hardtypes.DefaultGenesisState()
	hardGenState.Params.MoneyMarkets = []hardtypes.MoneyMarket{
		{
			Denom: "usdx",
			BorrowLimit: hardtypes.BorrowLimit{
				HasMaxLimit:  true,
				MaximumLimit: sdk.MustNewDecFromStr("100000000000"),
				LoanToValue:  sdk.MustNewDecFromStr("1"),
			},
			SpotMarketID:     "usdx:usd",
			ConversionFactor: sdkmath.NewInt(1_000_000),
			InterestRateModel: hardtypes.InterestRateModel{
				BaseRateAPY:    sdk.MustNewDecFromStr("0.05"),
				BaseMultiplier: sdk.MustNewDecFromStr("2"),
				Kink:           sdk.MustNewDecFromStr("0.8"),
				JumpMultiplier: sdk.MustNewDecFromStr("10"),
			},
			ReserveFactor:          sdk.MustNewDecFromStr("0.05"),
			KeeperRewardPercentage: sdk.ZeroDec(),
		},
	}

	pricefeedGenState := pricefeedtypes.DefaultGenesisState()
	pricefeedGenState.Params.Markets = []pricefeedtypes.Market{
		{MarketID: "usdx:usd", BaseAsset: "usdx", QuoteAsset: "usd", Oracles: []sdk.AccAddress{}, Active: true},
		{MarketID: "usdc:usd", BaseAsset: "usdc", QuoteAsset: "usd", Oracles: []sdk.AccAddress{}, Active: true},
		{MarketID: "usdc:usd:30", BaseAsset: "usdc", QuoteAsset: "usd", Oracles: []sdk.AccAddress{}, Active: true},
	}
	now := time.Now().Add(1 * time.Hour)
	pricefeedGenState.PostedPrices = []pricefeedtypes.PostedPrice{
		{MarketID: "usdx:usd", OracleAddress: sdk.AccAddress{}, Price: sdk.MustNewDecFromStr("1.00"), Expiry: now},
		{MarketID: "usdc:usd", OracleAddress: sdk.AccAddress{}, Price: sdk.MustNewDecFromStr("1.00"), Expiry: now},
		{MarketID: "usdc:usd:30", OracleAddress: sdk.AccAddress{}, Price: sdk.MustNewDecFromStr("1.00"), Expiry: now},
	}

	genState := app.GenesisState{
		evmtypes.ModuleName:       cdc.MustMarshalJSON(evmGs),
		feemarkettypes.ModuleName: cdc.MustMarshalJSON(feemarketGenesis),
		cdptypes.ModuleName:       cdc.MustMarshalJSON(&cdpGenState),
		hardtypes.ModuleName:      cdc.MustMarshalJSON(&hardGenState),
		pricefeedtypes.ModuleName: cdc.MustMarshalJSON(&pricefeedGenState),
	}

	// 资助测试账户
	coinsGenState := app.NewFundedGenStateWithSameCoins(
		tApp.AppCodec(),
		sdk.NewCoins(sdk.NewInt64Coin(gasDenom, 1e9)),
		[]sdk.AccAddress{suite.testAddr, suite.testAddr2},
	)

	tApp.InitializeFromGenesisStatesWithTimeAndChainID(
		time.Date(1998, 1, 1, 0, 0, 0, 0, time.UTC),
		ChainID,
		genState,
		coinsGenState,
	)

	// consensus key / header
	consPriv, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	consAddress := sdk.ConsAddress(consPriv.PubKey().Address())

	ctx := tApp.NewContext(false, tmproto.Header{
		Height:          tApp.LastBlockHeight() + 1,
		ChainID:         ChainID,
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),
		Version:         tmversion.Consensus{Block: version.BlockProtocol},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})
	suite.ctx = ctx

	// 设置 validator（EVM Config 依赖）
	valAcc := &etherminttypes.EthAccount{
		BaseAccount: authtypes.NewBaseAccount(sdk.AccAddress(consAddress.Bytes()), nil, 0, 0),
		CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
	}
	tApp.GetAccountKeeper().SetAccount(ctx, valAcc)
	_, testAddresses := app.GeneratePrivKeyAddressPairs(1)
	valAddr := sdk.ValAddress(testAddresses[0].Bytes())
	validator, err := stakingtypes.NewValidator(valAddr, consPriv.PubKey(), stakingtypes.Description{})
	suite.Require().NoError(err)
	suite.Require().NoError(tApp.GetStakingKeeper().SetValidatorByConsAddr(ctx, validator))
	tApp.GetStakingKeeper().SetValidator(ctx, validator)

	// 部署 USDC ERC20 合约 & 注册转换对
	contractAddr := suite.deployUSDCERC20()
	pair := evmutiltypes.NewConversionPair(contractAddr, USDCCoinDenom)
	suite.usdcEVMAddr = pair.GetAddress()

	// 提升块 Gas 上限
	cParams := tApp.GetConsensusParams(suite.ctx)
	cParams.Block.MaxGas = sims.DefaultGenTxGas * 20
	tApp.StoreConsensusParams(suite.ctx, cParams)

	// 启用一个默认转换对（历史保留）
	evmutilParams := suite.evmutilKeeper.GetParams(suite.ctx)
	evmutilParams.EnabledConversionPairs = evmutiltypes.NewConversionPairs(
		evmutiltypes.NewConversionPair(
			evmutiltestutil.MustNewInternalEVMAddressFromString("0x15932E26f5BD4923d46a2b205191C4b5d5f43FE3"),
			USDCCoinDenom,
		),
	)
	suite.evmutilKeeper.SetParams(suite.ctx, evmutilParams)

	// 允许通过 EIP712 的消息列表
	evmKeeper := suite.tApp.GetEvmKeeper()
	params := evmKeeper.GetParams(suite.ctx)
	params.EIP712AllowedMsgs = []evmtypes.EIP712AllowedMsg{
		{
			MsgTypeUrl:       "/istchain.evmutil.v1beta1.MsgConvertERC20ToCoin",
			MsgValueTypeName: "MsgValueEVMConvertERC20ToCoin",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "initiator", Type: "string"},
				{Name: "receiver", Type: "string"},
				{Name: "istchain_erc20_address", Type: "string"},
				{Name: "amount", Type: "string"},
			},
		},
		{
			MsgTypeUrl:       "/istchain.cdp.v1beta1.MsgCreateCDP",
			MsgValueTypeName: "MsgValueCDPCreate",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "sender", Type: "string"},
				{Name: "collateral", Type: "Coin"},
				{Name: "principal", Type: "Coin"},
				{Name: "collateral_type", Type: "string"},
			},
		},
		{
			MsgTypeUrl:       "/istchain.cdp.v1beta1.MsgDeposit",
			MsgValueTypeName: "MsgValueCDPDeposit",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "depositor", Type: "string"},
				{Name: "owner", Type: "string"},
				{Name: "collateral", Type: "Coin"},
				{Name: "collateral_type", Type: "string"},
			},
		},
		{
			MsgTypeUrl:       "/istchain.hard.v1beta1.MsgDeposit",
			MsgValueTypeName: "MsgValueHardDeposit",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "depositor", Type: "string"},
				{Name: "amount", Type: "Coin[]"},
			},
		},
		{
			MsgTypeUrl:       "/istchain.evmutil.v1beta1.MsgConvertCoinToERC20",
			MsgValueTypeName: "MsgValueEVMConvertCoinToERC20",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "initiator", Type: "string"},
				{Name: "receiver", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		{
			MsgTypeUrl:       "/istchain.cdp.v1beta1.MsgRepayDebt",
			MsgValueTypeName: "MsgValueCDPRepayDebt",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "sender", Type: "string"},
				{Name: "collateral_type", Type: "string"},
				{Name: "payment", Type: "Coin"},
			},
		},
		{
			MsgTypeUrl:       "/istchain.hard.v1beta1.MsgWithdraw",
			MsgValueTypeName: "MsgValueHardWithdraw",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "depositor", Type: "string"},
				{Name: "amount", Type: "Coin[]"},
			},
		},
	}
	suite.Require().NoError(evmKeeper.SetParams(suite.ctx, params))

	// 初始发 50k USDC（ERC20）
	initBal := suite.getEVMAmount(50_000)
	suite.Require().NoError(suite.evmutilKeeper.MintERC20(ctx, pair.GetAddress(), suite.testEVMAddr, initBal.BigInt()))
	suite.Require().NoError(suite.evmutilKeeper.MintERC20(ctx, pair.GetAddress(), suite.testEVMAddr2, initBal.BigInt()))

	// 触发 feemarket beginblock 设置 basefee
	suite.Commit()
	suite.tApp.GetFeeMarketKeeper().SetBaseFee(suite.ctx, big.NewInt(100))
}

func (suite *EIP712TestSuite) Commit() {
	_ = suite.tApp.Commit()
	header := suite.ctx.BlockHeader()
	header.Height++
	suite.tApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	suite.ctx = suite.tApp.NewContext(false, header)
}

func (suite *EIP712TestSuite) deployUSDCERC20() evmutiltypes.InternalEVMAddress {
	suite.Require().NoError(
		suite.tApp.FundModuleAccount(suite.ctx, evmutiltypes.ModuleName, sdk.NewCoins(sdk.NewCoin(gasDenom, sdk.ZeroInt()))),
	)
	addr, err := suite.evmutilKeeper.DeployTestMintableERC20Contract(suite.ctx, "USDC", "USDC", uint8(18))
	suite.Require().NoError(err)
	suite.Require().Greater(len(addr.Address), 0)
	return addr
}

// ---- tests ------------------------------------------------------------------

func (suite *EIP712TestSuite) TestEIP712Tx() {
	testcases := []struct {
		name           string
		usdcDepositAmt int64
		usdxToMintAmt  int64
		updateTx       func(txBuilder client.TxBuilder, msgs []sdk.Msg) client.TxBuilder
		updateMsgs     func(msgs []sdk.Msg) []sdk.Msg
		failCheckTx    bool
		errMsg         string
	}{
		{
			name:           "processes deposit eip712 messages successfully",
			usdcDepositAmt: 100, usdxToMintAmt: 99,
		},
		{
			name:           "fails when conversion more erc20 usdc than balance",
			usdcDepositAmt: 51_000, usdxToMintAmt: 100,
			errMsg: "transfer amount exceeds balance",
		},
		{
			name:           "fails when minting more usdx than allowed",
			usdcDepositAmt: 100, usdxToMintAmt: 100,
			errMsg: "proposed collateral ratio is below liquidation ratio",
		},
		{
			name:           "fails when trying to convert usdc for another address",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			errMsg:      "unauthorized",
			failCheckTx: true,
			updateMsgs: func(msgs []sdk.Msg) []sdk.Msg {
				convertMsg := evmutiltypes.NewMsgConvertERC20ToCoin(
					suite.testEVMAddr2, suite.testAddr, suite.usdcEVMAddr, suite.getEVMAmount(100),
				)
				msgs[0] = &convertMsg
				return msgs
			},
		},
		{
			name:           "fails when trying to convert erc20 for non-whitelisted contract",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			errMsg: "ERC20 token not enabled to convert to sdk.Coin",
			updateMsgs: func(msgs []sdk.Msg) []sdk.Msg {
				convertMsg := evmutiltypes.NewMsgConvertERC20ToCoin(
					suite.testEVMAddr, suite.testAddr, suite.testEVMAddr2, suite.getEVMAmount(100),
				)
				msgs[0] = &convertMsg
				return msgs
			},
		},
		{
			name:           "fails when signer tries to send messages with invalid signature",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			failCheckTx: true,
			errMsg:      "tx intended signer does not match the given signer",
			updateTx: func(txBuilder client.TxBuilder, _ []sdk.Msg) client.TxBuilder {
				option, err := codectypes.NewAnyWithValue(&etherminttypes.ExtensionOptionsWeb3Tx{
					FeePayer:         suite.testAddr.String(),
					TypedDataChainID: 2221,
					FeePayerSig:      []byte("sig"),
				})
				suite.Require().NoError(err)
				builder, _ := txBuilder.(authtx.ExtensionOptionsTxBuilder)
				builder.SetExtensionOptions(option)
				return txBuilder
			},
		},
		{
			name:           "fails when insufficient gas fees are provided",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			errMsg: "insufficient funds",
			updateTx: func(txBuilder client.TxBuilder, _ []sdk.Msg) client.TxBuilder {
				bk := suite.tApp.GetBankKeeper()
				gasCoins := bk.GetBalance(suite.ctx, suite.testAddr, gasDenom)
				suite.Require().NoError(
					bk.SendCoins(suite.ctx, suite.testAddr, suite.testAddr2, sdk.NewCoins(gasCoins)),
				)
				return txBuilder
			},
		},
		{
			name:           "fails when invalid chain id is provided",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			failCheckTx: true, errMsg: "invalid chain-id",
			updateTx: func(_ client.TxBuilder, msgs []sdk.Msg) client.TxBuilder {
				return suite.createTestEIP712CosmosTxBuilder(
					suite.testAddr, suite.testPrivKey, "isttest_12-1", suite.gasCoins(20), msgs,
				)
			},
		},
		{
			name:           "fails when invalid fee payer is provided",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			failCheckTx: true, errMsg: "invalid pubkey",
			updateTx: func(_ client.TxBuilder, msgs []sdk.Msg) client.TxBuilder {
				return suite.createTestEIP712CosmosTxBuilder(
					suite.testAddr2, suite.testPrivKey2, ChainID, suite.gasCoins(20), msgs,
				)
			},
		},
		{
			name:           "passes when domain matches expected fields",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			failCheckTx: false, errMsg: "",
			updateTx: func(_ client.TxBuilder, msgs []sdk.Msg) client.TxBuilder {
				pc, err := etherminttypes.ParseChainID(ChainID)
				suite.Require().NoError(err)
				domain, domainTypes := domainExpected(pc.Int64())
				return suite.createTestEIP712CosmosTxBuilderWithDomain(
					suite.testAddr, suite.testPrivKey, ChainID, suite.gasCoins(20), msgs, &domain, domainTypes,
				)
			},
		},
		{
			name:           "fails when domain.verifyingContract is non-empty string type",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			failCheckTx: true, errMsg: "signature verification failed",
			updateTx: func(_ client.TxBuilder, msgs []sdk.Msg) client.TxBuilder {
				pc, err := etherminttypes.ParseChainID(ChainID)
				suite.Require().NoError(err)
				domain := apitypes.TypedDataDomain{
					Name:    domainName,
					Version: domainVersion,
					ChainId: math.NewHexOrDecimal256(pc.Int64()),
					Salt:    "",
					// 错误: 非空 string
					VerifyingContract: "istchainCosmos",
				}
				domainTypes := []apitypes.Type{
					{Name: "name", Type: "string"},
					{Name: "version", Type: "string"},
					{Name: "chainId", Type: "uint256"},
					{Name: "verifyingContract", Type: "string"},
				}
				return suite.createTestEIP712CosmosTxBuilderWithDomain(
					suite.testAddr, suite.testPrivKey, ChainID, suite.gasCoins(20), msgs, &domain, domainTypes,
				)
			},
		},
		{
			name:           "fails when domain.verifyingContract is non-empty address type",
			usdcDepositAmt: 100, usdxToMintAmt: 90,
			failCheckTx: true, errMsg: "signature verification failed",
			updateTx: func(_ client.TxBuilder, msgs []sdk.Msg) client.TxBuilder {
				pc, err := etherminttypes.ParseChainID(ChainID)
				suite.Require().NoError(err)
				domain := apitypes.TypedDataDomain{
					Name:              domainName,
					Version:           domainVersion,
					ChainId:           math.NewHexOrDecimal256(pc.Int64()),
					Salt:              "",
					VerifyingContract: "0xc6d953c98f260cb7c3667cac87e5d508a0a81277",
				}
				domainTypes := []apitypes.Type{
					{Name: "name", Type: "string"},
					{Name: "version", Type: "string"},
					{Name: "chainId", Type: "uint256"},
					{Name: "verifyingContract", Type: "address"},
				}
				return suite.createTestEIP712CosmosTxBuilderWithDomain(
					suite.testAddr, suite.testPrivKey, ChainID, suite.gasCoins(20), msgs, &domain, domainTypes,
				)
			},
		},
	}

	for _, tc := range testcases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			msgs := suite.depositMintLendMsgs(tc.usdcDepositAmt, tc.usdxToMintAmt)
			if tc.updateMsgs != nil {
				msgs = tc.updateMsgs(msgs)
			}

			txBuilder := suite.createTestEIP712CosmosTxBuilder(
				suite.testAddr, suite.testPrivKey, ChainID, suite.gasCoins(20), msgs,
			)
			if tc.updateTx != nil {
				txBuilder = tc.updateTx(txBuilder, msgs)
			}

			txBytes := suite.mustEncodeTx(txBuilder)

			resCheckTx := suite.tApp.CheckTx(abci.RequestCheckTx{Tx: txBytes, Type: abci.CheckTxType_New})
			if !tc.failCheckTx {
				suite.Require().Equal(uint32(0), resCheckTx.Code, resCheckTx.Log)
			} else {
				suite.Require().NotEqual(uint32(0), resCheckTx.Code, resCheckTx.Log)
				suite.Require().Contains(resCheckTx.Log, tc.errMsg)
			}

			resDeliverTx := suite.tApp.DeliverTx(abci.RequestDeliverTx{Tx: txBytes})
			if tc.errMsg == "" {
				suite.Require().Equal(uint32(0), resDeliverTx.Code, resDeliverTx.Log)

				// user cosmos erc20/usdc balance
				bk := suite.tApp.GetBankKeeper()
				amt := bk.GetBalance(suite.ctx, suite.testAddr, USDCCoinDenom)
				suite.Require().Equal(sdk.ZeroInt(), amt.Amount)

				// cdp
				usdxAmt := sdkmath.NewInt(1_000_000).Mul(sdkmath.NewInt(tc.usdxToMintAmt))
				cdp, found := suite.tApp.GetCDPKeeper().GetCdpByOwnerAndCollateralType(suite.ctx, suite.testAddr, USDCCDPType)
				suite.Require().True(found)
				suite.Require().Equal(suite.testAddr, cdp.Owner)
				suite.Require().Equal(sdk.NewCoin(USDCCoinDenom, suite.getEVMAmount(100)), cdp.Collateral)
				suite.Require().Equal(sdk.NewCoin(stableDenom, usdxAmt), cdp.Principal)

				// hard
				hardDeposit, found := suite.tApp.GetHardKeeper().GetDeposit(suite.ctx, suite.testAddr)
				suite.Require().True(found)
				suite.Require().Equal(suite.testAddr, hardDeposit.Depositor)
				suite.Require().Equal(sdk.NewCoins(sdk.NewCoin(stableDenom, usdxAmt)), hardDeposit.Amount)
			} else {
				suite.Require().NotEqual(uint32(0), resDeliverTx.Code, resCheckTx.Log)
				suite.Require().Contains(resDeliverTx.Log, tc.errMsg)
			}
		})
	}
}

func (suite *EIP712TestSuite) TestEIP712Tx_DepositAndWithdraw() {
	// deposit flow
	depositMsgs := suite.depositMintLendMsgs(100, 99)
	txBuilder := suite.createTestEIP712CosmosTxBuilder(
		suite.testAddr, suite.testPrivKey, ChainID, suite.gasCoins(20), depositMsgs,
	)
	txBytes := suite.mustEncodeTx(txBuilder)
	resDeliverTx := suite.tApp.DeliverTx(abci.RequestDeliverTx{Tx: txBytes})
	suite.Require().Equal(uint32(0), resDeliverTx.Code, resDeliverTx.Log)

	// validate hard
	hardDeposit, found := suite.tApp.GetHardKeeper().GetDeposit(suite.ctx, suite.testAddr)
	suite.Require().True(found)
	suite.Require().Equal(suite.testAddr, hardDeposit.Depositor)
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin(stableDenom, sdkmath.NewInt(99_000_000))), hardDeposit.Amount)

	// erc20 balance
	coinBal, err := suite.evmutilKeeper.QueryERC20BalanceOf(suite.ctx, suite.usdcEVMAddr, suite.testEVMAddr)
	suite.Require().NoError(err)
	suite.Require().Equal(suite.getEVMAmount(49_900).BigInt(), coinBal)

	// withdraw flow
	usdcAmt := suite.getEVMAmount(100)
	usdxAmt := sdkmath.NewInt(99_000_000)

	withdrawConvertMsg := evmutiltypes.NewMsgConvertCoinToERC20(
		suite.testAddr.String(), suite.testEVMAddr.String(), sdk.NewCoin(USDCCoinDenom, usdcAmt),
	)
	cdpWithdrawMsg := cdptypes.NewMsgRepayDebt(suite.testAddr, USDCCDPType, sdk.NewCoin(stableDenom, usdxAmt))
	hardWithdrawMsg := hardtypes.NewMsgWithdraw(suite.testAddr, sdk.NewCoins(sdk.NewCoin(stableDenom, usdxAmt)))

	withdrawMsgs := []sdk.Msg{&hardWithdrawMsg, &cdpWithdrawMsg, &withdrawConvertMsg}
	txBuilder = suite.createTestEIP712CosmosTxBuilder(
		suite.testAddr, suite.testPrivKey, ChainID, suite.gasCoins(20), withdrawMsgs,
	)
	txBytes = suite.mustEncodeTx(txBuilder)
	resDeliverTx = suite.tApp.DeliverTx(abci.RequestDeliverTx{Tx: txBytes})
	suite.Require().Equal(uint32(0), resDeliverTx.Code, resDeliverTx.Log)

	// post conditions
	_, found = suite.tApp.GetHardKeeper().GetDeposit(suite.ctx, suite.testAddr)
	suite.Require().False(found)
	_, found = suite.tApp.GetCDPKeeper().GetCdpByOwnerAndCollateralType(suite.ctx, suite.testAddr, USDCCDPType)
	suite.Require().False(found)

	// user cosmos erc20/usdc balance should be zero
	bk := suite.tApp.GetBankKeeper()
	amt := bk.GetBalance(suite.ctx, suite.testAddr, USDCCoinDenom)
	suite.Require().Equal(sdk.ZeroInt(), amt.Amount)

	coinBal, err = suite.evmutilKeeper.QueryERC20BalanceOf(suite.ctx, suite.usdcEVMAddr, suite.testEVMAddr)
	suite.Require().NoError(err)
	suite.Require().Equal(suite.getEVMAmount(50_000).BigInt(), coinBal)
}

func TestEIP712Suite(t *testing.T) {
	suite.Run(t, new(EIP712TestSuite))
}
