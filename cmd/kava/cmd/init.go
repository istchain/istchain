package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	tmrand "github.com/cometbft/cometbft/libs/rand"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	genutil "github.com/cosmos/cosmos-sdk/x/genutil"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	// 若无 EVM 模块，可删除这两行
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

const (
	FlagOverwrite        = "overwrite"
	FlagRecover          = "recover"
	FlagDefaultBondDenom = "default-denom"
)

// InitCmd initializes a new genesis file with customized default parameters (uist denom).
func InitCmd(mbm module.BasicManager, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize node validator, config, and genesis.json",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec

			serverCtx := server.GetServerContextFromCmd(cmd)
			cfg := serverCtx.Config
			cfg.SetRoot(clientCtx.HomeDir)

			// chain-id
			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			switch {
			case chainID != "":
			case clientCtx.ChainID != "":
				chainID = clientCtx.ChainID
			default:
				chainID = fmt.Sprintf("istchain-%v", tmrand.Str(6))
			}

			// recover from mnemonic if provided
			var mnemonic string
			recover, _ := cmd.Flags().GetBool(FlagRecover)
			if recover {
				inBuf := bufio.NewReader(cmd.InOrStdin())
				value, err := input.GetString("Enter your bip39 mnemonic", inBuf)
				if err != nil {
					return err
				}
				mnemonic = value
				if !bip39.IsMnemonicValid(mnemonic) {
					return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid mnemonic")
				}
			}

			// initial height
			initHeight, _ := cmd.Flags().GetInt64(flags.FlagInitHeight)
			if initHeight < 1 {
				initHeight = 1
			}

			nodeID, _, err := genutil.InitializeNodeValidatorFilesFromMnemonic(cfg, mnemonic)
			if err != nil {
				return err
			}
			cfg.Moniker = args[0]

			genFile := cfg.GenesisFile()
			overwrite, _ := cmd.Flags().GetBool(FlagOverwrite)
			if _, err := os.Stat(genFile); err == nil && !overwrite {
				return fmt.Errorf("genesis.json already exists: %s", genFile)
			}

			// 生成默认 genesis state
			appGenState := mbm.DefaultGenesis(cdc)

			// -------- 仅改必要字段：统一 denom => uist --------

			// staking
			{
				var stakingGen stakingtypes.GenesisState
				cdc.MustUnmarshalJSON(appGenState[stakingtypes.ModuleName], &stakingGen)
				stakingGen.Params.BondDenom = "uist"
				appGenState[stakingtypes.ModuleName] = cdc.MustMarshalJSON(&stakingGen)
			}

			// mint
			{
				var mintGen minttypes.GenesisState
				cdc.MustUnmarshalJSON(appGenState[minttypes.ModuleName], &mintGen)
				mintGen.Params.MintDenom = "uist"
				appGenState[minttypes.ModuleName] = cdc.MustMarshalJSON(&mintGen)
			}

			// gov —— 同时兼容 v1 与 v1beta1
			if bz, ok := appGenState[govtypes.ModuleName]; ok {
				// 先看是不是 v1（是否有 "params" 键）
				var raw map[string]json.RawMessage
				_ = json.Unmarshal(bz, &raw)

				if _, hasParams := raw["params"]; hasParams {
					// ----- v1 -----
					var gs govv1.GenesisState
					if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
						return fmt.Errorf("failed to parse gov v1 genesis: %w", err)
					}
					if gs.Params == nil {
						gs.Params = &govv1.Params{}
					}
					if len(gs.Params.MinDeposit) == 0 {
						gs.Params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin("uist", 10_000_000)) // 10 IST
					} else {
						for i := range gs.Params.MinDeposit {
							gs.Params.MinDeposit[i].Denom = "uist"
						}
					}
					if gs.StartingProposalId == 0 {
						gs.StartingProposalId = 1
					}
					appGenState[govtypes.ModuleName] = cdc.MustMarshalJSON(&gs)
				} else {
					// ----- v1beta1 -----
					var gs govv1beta1.GenesisState
					if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
						return fmt.Errorf("failed to parse gov v1beta1 genesis: %w", err)
					}
					if len(gs.DepositParams.MinDeposit) == 0 {
						gs.DepositParams.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin("uist", 10_000_000))
					} else {
						for i := range gs.DepositParams.MinDeposit {
							gs.DepositParams.MinDeposit[i].Denom = "uist"
						}
					}
					if gs.StartingProposalId == 0 {
						gs.StartingProposalId = 1
					}
					appGenState[govtypes.ModuleName] = cdc.MustMarshalJSON(&gs)
				}
			}

			// crisis
			{
				var crisisGen crisistypes.GenesisState
				cdc.MustUnmarshalJSON(appGenState[crisistypes.ModuleName], &crisisGen)
				crisisGen.ConstantFee = sdk.NewInt64Coin("uist", 1000)
				appGenState[crisistypes.ModuleName] = cdc.MustMarshalJSON(&crisisGen)
			}

			// bank metadata
			{
				var bankGen banktypes.GenesisState
				cdc.MustUnmarshalJSON(appGenState[banktypes.ModuleName], &bankGen)
				bankGen.DenomMetadata = []banktypes.Metadata{
					{
						Base:    "uist",
						Display: "IST",
						Name:    "IST",
						Symbol:  "IST",
						DenomUnits: []*banktypes.DenomUnit{
							{Denom: "uist", Exponent: 0},
							{Denom: "IST", Exponent: 6},
						},
					},
				}
				appGenState[banktypes.ModuleName] = cdc.MustMarshalJSON(&bankGen)
			}

			// EVM（如包含）
			if bz, ok := appGenState[evmtypes.ModuleName]; ok {
				var evmGen evmtypes.GenesisState
				cdc.MustUnmarshalJSON(bz, &evmGen)
				evmGen.Params.EvmDenom = "uist"
				appGenState[evmtypes.ModuleName] = cdc.MustMarshalJSON(&evmGen)
			}

			// -------- 结束最小化修改 --------

			appState, err := json.MarshalIndent(appGenState, "", " ")
			if err != nil {
				return sdkerrors.Wrap(err, "failed to marshal app genesis state")
			}

			genDoc := &tmtypes.GenesisDoc{
				ChainID:       chainID,
				Validators:    nil,
				AppState:      appState,
				InitialHeight: initHeight,
			}
			if err := genutil.ExportGenesisFile(genDoc, genFile); err != nil {
				return sdkerrors.Wrap(err, "failed to write genesis file")
			}

			fmt.Printf(
				"Initialized istchain node\nMoniker: %s\nChainID: %s\nNodeID: %s\n",
				cfg.Moniker, chainID, nodeID,
			)
			return nil
		},
	}

	// flags
	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "node home directory")
	cmd.Flags().Bool(FlagOverwrite, false, "overwrite genesis.json")
	cmd.Flags().Bool(FlagRecover, false, "recover from mnemonic")
	cmd.Flags().String(flags.FlagChainID, "", "custom chain-id")
	cmd.Flags().Int64(flags.FlagInitHeight, 1, "initial block height")

	return cmd
}
