package keeper

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/istchain/istchain/x/auction/types"
)

// StartSurplusAuction starts a new surplus (forward) auction.
func (k Keeper) StartSurplusAuction(ctx sdk.Context, seller string, lot sdk.Coin, bidDenom string) (uint64, error) {
	auction := types.NewSurplusAuction(
		seller,
		lot,
		bidDenom,
		types.DistantFuture,
	)

	// Auction module holds the lot during the auction.
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, seller, types.ModuleName, sdk.NewCoins(lot)); err != nil {
		return 0, fmt.Errorf("move lot to auction module failed: %w", err)
	}

	auctionID, err := k.StoreNewAuction(ctx, &auction)
	if err != nil {
		return 0, fmt.Errorf("store auction failed: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionStart,
			sdk.NewAttribute(types.AttributeKeyAuctionID, fmt.Sprintf("%d", auctionID)),
			sdk.NewAttribute(types.AttributeKeyAuctionType, auction.GetType()),
			sdk.NewAttribute(types.AttributeKeyBid, auction.Bid.String()),
			sdk.NewAttribute(types.AttributeKeyLot, auction.Lot.String()),
		),
	)

	return auctionID, nil
}

// StartDebtAuction starts a new debt (reverse) auction.
func (k Keeper) StartDebtAuction(ctx sdk.Context, buyer string, bid sdk.Coin, initialLot sdk.Coin, debt sdk.Coin) (uint64, error) {
	auction := types.NewDebtAuction(
		buyer,
		bid,
		initialLot,
		types.DistantFuture,
		debt,
	)

	// Ensure initiator module can mint to avoid end-block errors.
	macc := k.accountKeeper.GetModuleAccount(ctx, buyer)
	if macc == nil {
		return 0, fmt.Errorf("module account %q not found", buyer)
	}
	if !macc.HasPermission(authtypes.Minter) {
		return 0, fmt.Errorf("module %q lacks %q permission", buyer, authtypes.Minter)
	}

	// Auction module holds the debt during the auction.
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, buyer, types.ModuleName, sdk.NewCoins(debt)); err != nil {
		return 0, fmt.Errorf("move debt to auction module failed: %w", err)
	}

	auctionID, err := k.StoreNewAuction(ctx, &auction)
	if err != nil {
		return 0, fmt.Errorf("store auction failed: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionStart,
			sdk.NewAttribute(types.AttributeKeyAuctionID, fmt.Sprintf("%d", auctionID)),
			sdk.NewAttribute(types.AttributeKeyAuctionType, auction.GetType()),
			sdk.NewAttribute(types.AttributeKeyBid, auction.Bid.String()),
			sdk.NewAttribute(types.AttributeKeyLot, auction.Lot.String()),
		),
	)

	return auctionID, nil
}

// StartCollateralAuction starts a new collateral (2-phase) auction.
func (k Keeper) StartCollateralAuction(
	ctx sdk.Context, seller string, lot, maxBid sdk.Coin,
	lotReturnAddrs []sdk.AccAddress, lotReturnWeights []sdkmath.Int, debt sdk.Coin,
) (uint64, error) {
	weightedAddresses, err := types.NewWeightedAddresses(lotReturnAddrs, lotReturnWeights)
	if err != nil {
		return 0, fmt.Errorf("build weighted addresses failed: %w", err)
	}

	auction := types.NewCollateralAuction(
		seller,
		lot,
		types.DistantFuture,
		maxBid,
		weightedAddresses,
		debt,
	)

	// Auction module holds both lot and debt during the auction.
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, seller, types.ModuleName, sdk.NewCoins(lot)); err != nil {
		return 0, fmt.Errorf("move lot to auction module failed: %w", err)
	}
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, seller, types.ModuleName, sdk.NewCoins(debt)); err != nil {
		return 0, fmt.Errorf("move debt to auction module failed: %w", err)
	}

	auctionID, err := k.StoreNewAuction(ctx, &auction)
	if err != nil {
		return 0, fmt.Errorf("store auction failed: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionStart,
			sdk.NewAttribute(types.AttributeKeyAuctionID, fmt.Sprintf("%d", auctionID)),
			sdk.NewAttribute(types.AttributeKeyAuctionType, auction.GetType()),
			sdk.NewAttribute(types.AttributeKeyBid, auction.Bid.String()),
			sdk.NewAttribute(types.AttributeKeyLot, auction.Lot.String()),
			sdk.NewAttribute(types.AttributeKeyMaxBid, auction.MaxBid.String()),
		),
	)

	return auctionID, nil
}

// PlaceBid places a bid on any auction.
func (k Keeper) PlaceBid(ctx sdk.Context, auctionID uint64, bidder sdk.AccAddress, newAmount sdk.Coin) error {
	auction, found := k.GetAuction(ctx, auctionID)
	if !found {
		return errorsmod.Wrapf(types.ErrAuctionNotFound, "%d", auctionID)
	}

	now := ctx.BlockTime()
	if now.After(auction.GetEndTime()) {
		return errorsmod.Wrapf(types.ErrAuctionHasExpired, "%d", auctionID)
	}

	var (
		err            error
		updatedAuction types.Auction
	)

	switch a := auction.(type) {
	case *types.SurplusAuction:
		updatedAuction, err = k.PlaceBidSurplus(ctx, a, bidder, newAmount)
	case *types.DebtAuction:
		updatedAuction, err = k.PlaceBidDebt(ctx, a, bidder, newAmount)
	case *types.CollateralAuction:
		if !a.IsReversePhase() {
			updatedAuction, err = k.PlaceForwardBidCollateral(ctx, a, bidder, newAmount)
		} else {
			updatedAuction, err = k.PlaceReverseBidCollateral(ctx, a, bidder, newAmount)
		}
	default:
		err = errorsmod.Wrap(types.ErrUnrecognizedAuctionType, auction.GetType())
	}

	if err != nil {
		return err
	}

	k.SetAuction(ctx, updatedAuction)
	return nil
}

// PlaceBidSurplus places a forward bid on a surplus auction.
func (k Keeper) PlaceBidSurplus(ctx sdk.Context, auction *types.SurplusAuction, bidder sdk.AccAddress, bid sdk.Coin) (*types.SurplusAuction, error) {
	if bid.Denom != auction.Bid.Denom {
		return auction, errorsmod.Wrapf(types.ErrInvalidBidDenom, "%s ≠ %s", bid.Denom, auction.Bid.Denom)
	}

	params := k.GetParams(ctx)
	minNewBidAmt := auction.Bid.Amount.Add(
		sdk.MaxInt(
			sdkmath.NewInt(1),
			sdk.NewDecFromInt(auction.Bid.Amount).Mul(params.IncrementSurplus).RoundInt(),
		),
	)
	if bid.Amount.LT(minNewBidAmt) {
		return auction, errorsmod.Wrapf(types.ErrBidTooSmall, "%s < %s%s", bid, minNewBidAmt, auction.Bid.Denom)
	}

	// Swap-out previous bidder if any (and not self-replace).
	if !bidder.Equals(auction.Bidder) && !auction.Bid.IsZero() {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, types.ModuleName, sdk.NewCoins(auction.Bid)); err != nil {
			return auction, fmt.Errorf("collect previous bid from new bidder failed: %w", err)
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, auction.Bidder, sdk.NewCoins(auction.Bid)); err != nil {
			return auction, fmt.Errorf("refund previous bidder failed: %w", err)
		}
	}

	// Bid increment goes to initiator and then burned.
	increment := bid.Sub(auction.Bid)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, auction.Initiator, sdk.NewCoins(increment)); err != nil {
		return auction, fmt.Errorf("send increment to initiator failed: %w", err)
	}
	if err := k.bankKeeper.BurnCoins(ctx, auction.Initiator, sdk.NewCoins(increment)); err != nil {
		return auction, fmt.Errorf("burn increment failed: %w", err)
	}

	// Update auction & timers.
	auction.Bidder = bidder
	auction.Bid = bid
	if !auction.HasReceivedBids {
		auction.MaxEndTime = ctx.BlockTime().Add(params.MaxAuctionDuration)
		auction.HasReceivedBids = true
	}
	auction.EndTime = earliestTime(ctx.BlockTime().Add(params.ForwardBidDuration), auction.MaxEndTime)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionBid,
			sdk.NewAttribute(types.AttributeKeyAuctionID, fmt.Sprintf("%d", auction.ID)),
			sdk.NewAttribute(types.AttributeKeyBidder, auction.Bidder.String()),
			sdk.NewAttribute(types.AttributeKeyBid, auction.Bid.String()),
			sdk.NewAttribute(types.AttributeKeyEndTime, fmt.Sprintf("%d", auction.EndTime.Unix())),
		),
	)
	return auction, nil
}

// PlaceForwardBidCollateral places a forward bid on a collateral auction.
func (k Keeper) PlaceForwardBidCollateral(ctx sdk.Context, auction *types.CollateralAuction, bidder sdk.AccAddress, bid sdk.Coin) (*types.CollateralAuction, error) {
	if bid.Denom != auction.Bid.Denom {
		return auction, errorsmod.Wrapf(types.ErrInvalidBidDenom, "%s ≠ %s", bid.Denom, auction.Bid.Denom)
	}
	if auction.IsReversePhase() {
		panic("cannot place forward bid on auction in reverse phase")
	}

	params := k.GetParams(ctx)
	minNewBidAmt := auction.Bid.Amount.Add(
		sdk.MaxInt(
			sdkmath.NewInt(1),
			sdk.NewDecFromInt(auction.Bid.Amount).Mul(params.IncrementCollateral).RoundInt(),
		),
	)
	// allow hitting MaxBid even if < increment threshold
	minNewBidAmt = sdk.MinInt(minNewBidAmt, auction.MaxBid.Amount)

	if bid.Amount.LT(minNewBidAmt) {
		return auction, errorsmod.Wrapf(types.ErrBidTooSmall, "%s < %s%s", bid, minNewBidAmt, auction.Bid.Denom)
	}
	if auction.MaxBid.IsLT(bid) {
		return auction, errorsmod.Wrapf(types.ErrBidTooLarge, "%s > %s", bid, auction.MaxBid)
	}

	// Swap-out previous bidder if applicable.
	if !bidder.Equals(auction.Bidder) && !auction.Bid.IsZero() {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, types.ModuleName, sdk.NewCoins(auction.Bid)); err != nil {
			return auction, fmt.Errorf("collect previous bid from new bidder failed: %w", err)
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, auction.Bidder, sdk.NewCoins(auction.Bid)); err != nil {
			return auction, fmt.Errorf("refund previous bidder failed: %w", err)
		}
	}

	increment := bid.Sub(auction.Bid)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, auction.Initiator, sdk.NewCoins(increment)); err != nil {
		return auction, fmt.Errorf("send increment to initiator failed: %w", err)
	}

	// Reduce corresponding debt up to the increment value.
	if auction.CorrespondingDebt.IsPositive() {
		debtAmountToReturn := sdk.MinInt(increment.Amount, auction.CorrespondingDebt.Amount)
		debtToReturn := sdk.NewCoin(auction.CorrespondingDebt.Denom, debtAmountToReturn)
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, auction.Initiator, sdk.NewCoins(debtToReturn)); err != nil {
			return auction, fmt.Errorf("return corresponding debt failed: %w", err)
		}
		auction.CorrespondingDebt = auction.CorrespondingDebt.Sub(debtToReturn)
	}

	// Update auction & timers.
	auction.Bidder = bidder
	auction.Bid = bid
	if !auction.HasReceivedBids {
		auction.MaxEndTime = ctx.BlockTime().Add(params.MaxAuctionDuration)
		auction.HasReceivedBids = true
	}
	// If the bid switches to reverse phase (protocol-specific), use reverse timer; otherwise forward.
	if auction.IsReversePhase() {
		auction.EndTime = earliestTime(ctx.BlockTime().Add(params.ReverseBidDuration), auction.MaxEndTime)
	} else {
		auction.EndTime = earliestTime(ctx.BlockTime().Add(params.ForwardBidDuration), auction.MaxEndTime)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionBid,
			sdk.NewAttribute(types.AttributeKeyAuctionID, fmt.Sprintf("%d", auction.ID)),
			sdk.NewAttribute(types.AttributeKeyBidder, auction.Bidder.String()),
			sdk.NewAttribute(types.AttributeKeyBid, auction.Bid.String()),
			sdk.NewAttribute(types.AttributeKeyEndTime, fmt.Sprintf("%d", auction.EndTime.Unix())),
		),
	)
	return auction, nil
}

// PlaceReverseBidCollateral places a reverse bid on a collateral auction.
func (k Keeper) PlaceReverseBidCollateral(ctx sdk.Context, auction *types.CollateralAuction, bidder sdk.AccAddress, lot sdk.Coin) (*types.CollateralAuction, error) {
	if lot.Denom != auction.Lot.Denom {
		return auction, errorsmod.Wrapf(types.ErrInvalidLotDenom, "%s ≠ %s", lot.Denom, auction.Lot.Denom)
	}
	if !auction.IsReversePhase() {
		panic("cannot place forward bid on auction in reverse phase")
	}

	params := k.GetParams(ctx)
	maxNewLotAmt := auction.Lot.Amount.Sub(
		sdk.MaxInt(
			sdkmath.NewInt(1),
			sdk.NewDecFromInt(auction.Lot.Amount).Mul(params.IncrementCollateral).RoundInt(),
		),
	)
	if lot.Amount.GT(maxNewLotAmt) {
		return auction, errorsmod.Wrapf(types.ErrLotTooLarge, "%s > %s%s", lot, maxNewLotAmt, auction.Lot.Denom)
	}
	if lot.IsNegative() {
		return auction, errorsmod.Wrapf(types.ErrLotTooSmall, "%s < 0%s", lot, auction.Lot.Denom)
	}

	// Swap-out previous bidder if applicable.
	if !bidder.Equals(auction.Bidder) {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, types.ModuleName, sdk.NewCoins(auction.Bid)); err != nil {
			return auction, fmt.Errorf("collect previous bid from new bidder failed: %w", err)
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, auction.Bidder, sdk.NewCoins(auction.Bid)); err != nil {
			return auction, fmt.Errorf("refund previous bidder failed: %w", err)
		}
	}

	// Pay out the decrease in lot to weighted addresses.
	lotDecrease := auction.Lot.Sub(lot)
	lotPayouts, err := splitCoinIntoWeightedBuckets(lotDecrease, auction.LotReturns.Weights)
	if err != nil {
		return auction, fmt.Errorf("split lot payouts failed: %w", err)
	}
	for i, payout := range lotPayouts {
		if !payout.IsPositive() {
			continue
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, auction.LotReturns.Addresses[i], sdk.NewCoins(payout)); err != nil {
			return auction, fmt.Errorf("send lot payout failed: %w", err)
		}
	}

	// Update auction & timers.
	auction.Bidder = bidder
	auction.Lot = lot
	if !auction.HasReceivedBids {
		auction.MaxEndTime = ctx.BlockTime().Add(params.MaxAuctionDuration)
		auction.HasReceivedBids = true
	}
	auction.EndTime = earliestTime(ctx.BlockTime().Add(params.ReverseBidDuration), auction.MaxEndTime)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionBid,
			sdk.NewAttribute(types.AttributeKeyAuctionID, fmt.Sprintf("%d", auction.ID)),
			sdk.NewAttribute(types.AttributeKeyBidder, auction.Bidder.String()),
			sdk.NewAttribute(types.AttributeKeyLot, auction.Lot.String()),
			sdk.NewAttribute(types.AttributeKeyEndTime, fmt.Sprintf("%d", auction.EndTime.Unix())),
		),
	)
	return auction, nil
}

// PlaceBidDebt places a reverse bid on a debt auction.
func (k Keeper) PlaceBidDebt(ctx sdk.Context, auction *types.DebtAuction, bidder sdk.AccAddress, lot sdk.Coin) (*types.DebtAuction, error) {
	if lot.Denom != auction.Lot.Denom {
		return auction, errorsmod.Wrapf(types.ErrInvalidLotDenom, "%s ≠ %s", lot.Denom, auction.Lot.Denom)
	}

	params := k.GetParams(ctx)
	maxNewLotAmt := auction.Lot.Amount.Sub(
		sdk.MaxInt(
			sdkmath.NewInt(1),
			sdk.NewDecFromInt(auction.Lot.Amount).Mul(params.IncrementDebt).RoundInt(),
		),
	)
	if lot.Amount.GT(maxNewLotAmt) {
		return auction, errorsmod.Wrapf(types.ErrLotTooLarge, "%s > %s%s", lot, maxNewLotAmt, auction.Lot.Denom)
	}
	if lot.IsNegative() {
		return auction, errorsmod.Wrapf(types.ErrLotTooSmall, "%s ≤ %s%s", lot, sdk.ZeroInt(), auction.Lot.Denom)
	}

	// Swap-out previous bidder (first bid special-case).
	if !bidder.Equals(auction.Bidder) {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, types.ModuleName, sdk.NewCoins(auction.Bid)); err != nil {
			return auction, fmt.Errorf("collect previous bid from new bidder failed: %w", err)
		}
		oldBidder := auction.Bidder
		if oldBidder.Equals(authtypes.NewModuleAddress(auction.Initiator)) {
			if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, auction.Initiator, sdk.NewCoins(auction.Bid)); err != nil {
				return auction, fmt.Errorf("return first bid to initiator module failed: %w", err)
			}
		} else {
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, oldBidder, sdk.NewCoins(auction.Bid)); err != nil {
				return auction, fmt.Errorf("refund previous bidder failed: %w", err)
			}
		}
	}

	// On first bid, return debt to initiator up to Bid amount.
	if auction.Bidder.Equals(authtypes.NewModuleAddress(auction.Initiator)) {
		debtAmountToReturn := sdk.MinInt(auction.Bid.Amount, auction.CorrespondingDebt.Amount)
		debtToReturn := sdk.NewCoin(auction.CorrespondingDebt.Denom, debtAmountToReturn)
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, auction.Initiator, sdk.NewCoins(debtToReturn)); err != nil {
			return auction, fmt.Errorf("return corresponding debt on first bid failed: %w", err)
		}
		auction.CorrespondingDebt = auction.CorrespondingDebt.Sub(debtToReturn)
	}

	// Update auction & timers.
	auction.Bidder = bidder
	auction.Lot = lot
	if !auction.HasReceivedBids {
		auction.MaxEndTime = ctx.BlockTime().Add(params.MaxAuctionDuration)
		auction.HasReceivedBids = true
	}
	auction.EndTime = earliestTime(ctx.BlockTime().Add(params.ForwardBidDuration), auction.MaxEndTime)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionBid,
			sdk.NewAttribute(types.AttributeKeyAuctionID, fmt.Sprintf("%d", auction.ID)),
			sdk.NewAttribute(types.AttributeKeyBidder, auction.Bidder.String()),
			sdk.NewAttribute(types.AttributeKeyLot, auction.Lot.String()),
			sdk.NewAttribute(types.AttributeKeyEndTime, fmt.Sprintf("%d", auction.EndTime.Unix())),
		),
	)
	return auction, nil
}

// CloseAuction closes an auction and distributes funds to the highest bidder.
func (k Keeper) CloseAuction(ctx sdk.Context, auctionID uint64) error {
	auction, found := k.GetAuction(ctx, auctionID)
	if !found {
		return errorsmod.Wrapf(types.ErrAuctionNotFound, "%d", auctionID)
	}

	now := ctx.BlockTime()
	if now.Before(auction.GetEndTime()) {
		return errorsmod.Wrapf(
			types.ErrAuctionHasNotExpired,
			"block time %s, auction end time %s",
			now.UTC(), auction.GetEndTime().UTC(),
		)
	}

	var err error
	switch a := auction.(type) {
	case *types.SurplusAuction:
		err = k.PayoutSurplusAuction(ctx, a)
	case *types.DebtAuction:
		err = k.PayoutDebtAuction(ctx, a)
	case *types.CollateralAuction:
		err = k.PayoutCollateralAuction(ctx, a)
	default:
		err = errorsmod.Wrap(types.ErrUnrecognizedAuctionType, auction.GetType())
	}
	if err != nil {
		return err
	}

	k.DeleteAuction(ctx, auctionID)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionClose,
			sdk.NewAttribute(types.AttributeKeyAuctionID, fmt.Sprintf("%d", auctionID)),
			sdk.NewAttribute(types.AttributeKeyCloseBlock, fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)
	return nil
}

// PayoutDebtAuction pays out the proceeds for a debt auction, first minting the coins.
func (k Keeper) PayoutDebtAuction(ctx sdk.Context, auction *types.DebtAuction) error {
	// Mint coins to pay off debt.
	if err := k.bankKeeper.MintCoins(ctx, auction.Initiator, sdk.NewCoins(auction.Lot)); err != nil {
		// keep panic to surface invariant errors (permission/config)
		panic(fmt.Errorf("could not mint coins: %w", err))
	}
	// Send minted coins from initiator module to winning bidder.
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, auction.Initiator, auction.Bidder, sdk.NewCoins(auction.Lot)); err != nil {
		return fmt.Errorf("send minted coins to bidder failed: %w", err)
	}
	// Return remaining debt (if any) to initiator for management.
	if !auction.CorrespondingDebt.IsPositive() {
		return nil
	}
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, auction.Initiator, sdk.NewCoins(auction.CorrespondingDebt))
}

// PayoutSurplusAuction pays out the proceeds for a surplus auction.
func (k Keeper) PayoutSurplusAuction(ctx sdk.Context, auction *types.SurplusAuction) error {
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, auction.Bidder, sdk.NewCoins(auction.Lot))
}

// PayoutCollateralAuction pays out the proceeds for a collateral auction.
func (k Keeper) PayoutCollateralAuction(ctx sdk.Context, auction *types.CollateralAuction) error {
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, auction.Bidder, sdk.NewCoins(auction.Lot)); err != nil {
		return fmt.Errorf("send lot to bidder failed: %w", err)
	}
	if !auction.CorrespondingDebt.IsPositive() {
		return nil
	}
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, auction.Initiator, sdk.NewCoins(auction.CorrespondingDebt))
}

// CloseExpiredAuctions iterates over all auctions up to current block time and closes expired ones.
func (k Keeper) CloseExpiredAuctions(ctx sdk.Context) error {
	var err error
	k.IterateAuctionsByTime(ctx, ctx.BlockTime(), func(id uint64) (stop bool) {
		if cerr := k.CloseAuction(ctx, id); cerr != nil && !errorsmod.Is(cerr, types.ErrAuctionNotFound) {
			err = cerr
			return true // stop on first meaningful error
		}
		return false
	})
	return err
}

// earliestTime returns the earlier of two times.
func earliestTime(t1, t2 time.Time) time.Time {
	if t1.Before(t2) {
		return t1
	}
	return t2
}

// splitCoinIntoWeightedBuckets divides up some amount of coins according to weights.
func splitCoinIntoWeightedBuckets(coin sdk.Coin, buckets []sdkmath.Int) ([]sdk.Coin, error) {
	amounts := splitIntIntoWeightedBuckets(coin.Amount, buckets)
	result := make([]sdk.Coin, len(amounts))
	for i, a := range amounts {
		result[i] = sdk.NewCoin(coin.Denom, a)
	}
	return result, nil
}
