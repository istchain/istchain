package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/istchain/istchain/x/auction/types"
)

// GetQueryCmd returns the cli query commands for the auction module
func GetQueryCmd() *cobra.Command {
	auctionQueryCmd := &cobra.Command{
		Use:   types.ModuleName,
		Short: fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
	}
	auctionQueryCmd.SilenceUsage = true

	cmds := []*cobra.Command{
		GetCmdQueryParams(),
		GetCmdQueryAuction(),
		GetCmdQueryAuctions(),
	}

	for _, cmd := range cmds {
		flags.AddQueryFlagsToCmd(cmd)
	}

	auctionQueryCmd.AddCommand(cmds...)
	return auctionQueryCmd
}

// GetCmdQueryParams queries the auction module parameters
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: fmt.Sprintf("get the %s module parameters", types.ModuleName),
		Long:  "Get the current auction module parameters.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, queryClient, err := getQueryClient(cmd)
			if err != nil {
				return err
			}

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return fmt.Errorf("query params failed: %w", err)
			}
			return clientCtx.PrintProto(&res.Params)
		},
	}
	cmd.SilenceUsage = true
	return cmd
}

// GetCmdQueryAuction queries one auction in the store
func GetCmdQueryAuction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auction [auction-id]",
		Short: "get info about an auction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, queryClient, err := getQueryClient(cmd)
			if err != nil {
				return err
			}

			auctionID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid auction-id %q: %w", args[0], err)
			}

			res, err := queryClient.Auction(cmd.Context(), &types.QueryAuctionRequest{
				AuctionId: auctionID,
			})
			if err != nil {
				return fmt.Errorf("query auction %d failed: %w", auctionID, err)
			}

			return clientCtx.PrintProto(res)
		},
	}
	cmd.SilenceUsage = true
	return cmd
}

// Query auction flags
const (
	flagType  = "type"
	flagDenom = "denom"
	flagPhase = "phase"
	flagOwner = "owner"
)

// GetCmdQueryAuctions queries the auctions in the store
func GetCmdQueryAuctions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auctions",
		Short: "query auctions with optional filters",
		Long:  "Query for all paginated auctions that match optional filters.",
		Example: strings.Join([]string{
			fmt.Sprintf("  $ %s q %s auctions --type=(collateral|surplus|debt)", version.AppName, types.ModuleName),
			fmt.Sprintf("  $ %s q %s auctions --owner=istchain1hatdq32u5x4wnxrtv5wzjzmq49sxgjgsj0mffm", version.AppName, types.ModuleName),
			fmt.Sprintf("  $ %s q %s auctions --denom=bnb", version.AppName, types.ModuleName),
			fmt.Sprintf("  $ %s q %s auctions --phase=(forward|reverse)", version.AppName, types.ModuleName),
			fmt.Sprintf("  $ %s q %s auctions --page=2 --limit=100", version.AppName, types.ModuleName),
		}, "\n"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			auctionType, _ := cmd.Flags().GetString(flagType)
			owner, _ := cmd.Flags().GetString(flagOwner)
			denom, _ := cmd.Flags().GetString(flagDenom)
			phase, _ := cmd.Flags().GetString(flagPhase)

			// Normalize inputs (to-lower where applicable)
			auctionType = strings.ToLower(strings.TrimSpace(auctionType))
			phase = strings.ToLower(strings.TrimSpace(phase))
			owner = strings.TrimSpace(owner)
			denom = strings.TrimSpace(denom)

			// Validate filters
			if err := validateAuctionType(auctionType); err != nil {
				return err
			}
			if err := validateOwner(owner, auctionType); err != nil {
				return err
			}
			if err := validateDenom(denom); err != nil {
				return err
			}
			if err := validatePhase(phase, auctionType); err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return fmt.Errorf("read pagination flags failed: %w", err)
			}

			clientCtx, queryClient, err := getQueryClient(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryAuctionsRequest{
				Type:       auctionType,
				Owner:      owner,
				Denom:      denom,
				Phase:      phase,
				Pagination: pageReq,
			}

			res, err := queryClient.Auctions(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("query auctions failed: %w", err)
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.SilenceUsage = true

	flags.AddPaginationFlagsToCmd(cmd, "auctions")
	cmd.Flags().String(flagType, "", "(optional) filter by auction type, type: collateral, debt, surplus")
	cmd.Flags().String(flagOwner, "", "(optional) filter by collateral auction owner")
	cmd.Flags().String(flagDenom, "", "(optional) filter by auction denom")
	cmd.Flags().String(flagPhase, "", "(optional) filter by collateral auction phase, phase: forward/reverse")

	return cmd
}

// ------------------------ helpers ------------------------

func getQueryClient(cmd *cobra.Command) (client.Context, types.QueryClient, error) {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return client.Context{}, nil, fmt.Errorf("get client query context failed: %w", err)
	}
	return clientCtx, types.NewQueryClient(clientCtx), nil
}

func validateAuctionType(t string) error {
	if t == "" {
		return nil
	}
	switch t {
	case types.CollateralAuctionType, types.SurplusAuctionType, types.DebtAuctionType:
		return nil
	default:
		return fmt.Errorf("invalid auction type %q (allowed: %s|%s|%s)",
			t, types.CollateralAuctionType, types.SurplusAuctionType, types.DebtAuctionType)
	}
}

func validateOwner(owner, auctionType string) error {
	if owner == "" {
		return nil
	}
	if auctionType != types.CollateralAuctionType {
		return fmt.Errorf("cannot apply --owner flag to non-collateral auction type")
	}
	if _, err := sdk.AccAddressFromBech32(owner); err != nil {
		return fmt.Errorf("cannot parse address from auction owner %q: %w", owner, err)
	}
	return nil
}

func validateDenom(denom string) error {
	if denom == "" {
		return nil
	}
	if err := sdk.ValidateDenom(denom); err != nil {
		return fmt.Errorf("invalid denom %q: %w", denom, err)
	}
	return nil
}

func validatePhase(phase, auctionType string) error {
	if phase == "" {
		return nil
	}
	if auctionType != "" && auctionType != types.CollateralAuctionType {
		return fmt.Errorf("cannot apply --phase flag to non-collateral auction type")
	}
	switch phase {
	case types.ForwardAuctionPhase, types.ReverseAuctionPhase:
		return nil
	default:
		return fmt.Errorf("invalid auction phase %q (allowed: %s|%s)",
			phase, types.ForwardAuctionPhase, types.ReverseAuctionPhase)
	}
}

// keep explicit import to ensure context is used by cmd.Context()
var _ = context.Background
