package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/istchain/istchain/x/auction/types"
)

type Keeper struct {
	storeKey      storetypes.StoreKey
	cdc           codec.Codec
	paramSubspace paramtypes.Subspace
	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper
}

// NewKeeper returns a new auction keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	paramstore paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
) Keeper {
	if !paramstore.HasKeyTable() {
		paramstore = paramstore.WithKeyTable(types.ParamKeyTable())
	}
	return Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		paramSubspace: paramstore,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
	}
}

// ---------- internal store helpers (preserve existing key layout) ----------

func (k Keeper) nextIDStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.NextAuctionIDKey)
}

func (k Keeper) auctionStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.AuctionKeyPrefix)
}

func (k Keeper) byTimeStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.AuctionByTimeKeyPrefix)
}

// ---------- codec helpers ----------

// MustUnmarshalAuction attempts to decode and return an Auction object from raw bytes. It panics on error.
func (k Keeper) MustUnmarshalAuction(bz []byte) types.Auction {
	auction, err := k.UnmarshalAuction(bz)
	if err != nil {
		panic(fmt.Errorf("failed to decode auction: %w", err))
	}
	return auction
}

// MustMarshalAuction attempts to encode an Auction object and returns raw bytes. It panics on error.
func (k Keeper) MustMarshalAuction(auction types.Auction) []byte {
	bz, err := k.MarshalAuction(auction)
	if err != nil {
		panic(fmt.Errorf("failed to encode auction: %w", err))
	}
	return bz
}

// MarshalAuction protobuf-serializes an Auction interface.
func (k Keeper) MarshalAuction(auctionI types.Auction) ([]byte, error) {
	return k.cdc.MarshalInterface(auctionI)
}

// UnmarshalAuction returns an Auction interface from proto-encoded bytes.
func (k Keeper) UnmarshalAuction(bz []byte) (types.Auction, error) {
	var auc types.Auction
	return auc, k.cdc.UnmarshalInterface(bz, &auc)
}

// ---------- misc ----------

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ---------- next ID management ----------

// SetNextAuctionID stores an ID to be used for the next created auction.
func (k Keeper) SetNextAuctionID(ctx sdk.Context, id uint64) {
	store := k.nextIDStore(ctx)
	// 注意：这里沿用原先“前缀 + 子键”的布局，避免引入状态迁移
	store.Set(types.NextAuctionIDKey, types.Uint64ToBytes(id))
}

// GetNextAuctionID reads the next available global ID from store.
func (k Keeper) GetNextAuctionID(ctx sdk.Context) (uint64, error) {
	store := k.nextIDStore(ctx)
	bz := store.Get(types.NextAuctionIDKey)
	if bz == nil {
		return 0, types.ErrInvalidInitialAuctionID
	}
	return types.Uint64FromBytes(bz), nil
}

// IncrementNextAuctionID increments the next auction ID in the store by 1.
func (k Keeper) IncrementNextAuctionID(ctx sdk.Context) error {
	id, err := k.GetNextAuctionID(ctx)
	if err != nil {
		return err
	}
	k.SetNextAuctionID(ctx, id+1)
	return nil
}

// StoreNewAuction stores an auction, assigning a new ID.
func (k Keeper) StoreNewAuction(ctx sdk.Context, auction types.Auction) (uint64, error) {
	newAuctionID, err := k.GetNextAuctionID(ctx)
	if err != nil {
		return 0, err
	}
	auction = auction.WithID(newAuctionID)
	k.SetAuction(ctx, auction)

	if err := k.IncrementNextAuctionID(ctx); err != nil {
		return 0, err
	}
	return newAuctionID, nil
}

// ---------- CRUD for auctions & indexes ----------

// SetAuction puts the auction into the store, and updates by-time index.
func (k Keeper) SetAuction(ctx sdk.Context, auction types.Auction) {
	// If exists, remove old by-time index first.
	if existing, found := k.GetAuction(ctx, auction.GetID()); found {
		k.removeFromByTimeIndex(ctx, existing.GetEndTime(), existing.GetID())
	}

	store := k.auctionStore(ctx)
	store.Set(types.GetAuctionKey(auction.GetID()), k.MustMarshalAuction(auction))

	k.InsertIntoByTimeIndex(ctx, auction.GetEndTime(), auction.GetID())
}

// GetAuction gets an auction from the store.
func (k Keeper) GetAuction(ctx sdk.Context, auctionID uint64) (types.Auction, bool) {
	store := k.auctionStore(ctx)
	bz := store.Get(types.GetAuctionKey(auctionID))
	if bz == nil {
		var empty types.Auction
		return empty, false
	}
	return k.MustUnmarshalAuction(bz), true
}

// DeleteAuction removes an auction from the store, and any indexes.
func (k Keeper) DeleteAuction(ctx sdk.Context, auctionID uint64) {
	if auction, found := k.GetAuction(ctx, auctionID); found {
		k.removeFromByTimeIndex(ctx, auction.GetEndTime(), auctionID)
	}
	store := k.auctionStore(ctx)
	store.Delete(types.GetAuctionKey(auctionID))
}

// InsertIntoByTimeIndex adds an auction ID and end time into the by-time index.
func (k Keeper) InsertIntoByTimeIndex(ctx sdk.Context, endTime time.Time, auctionID uint64) {
	store := k.byTimeStore(ctx)
	store.Set(types.GetAuctionByTimeKey(endTime, auctionID), types.Uint64ToBytes(auctionID))
}

// removeFromByTimeIndex removes an auction ID and end time from the by-time index.
func (k Keeper) removeFromByTimeIndex(ctx sdk.Context, endTime time.Time, auctionID uint64) {
	store := k.byTimeStore(ctx)
	store.Delete(types.GetAuctionByTimeKey(endTime, auctionID))
}

// ---------- iteration ----------

// IterateAuctionsByTime provides an iterator over auctions ordered by auction.EndTime (<= inclusiveCutoffTime).
// For each auction, cb will be called. If cb returns true, iteration stops.
func (k Keeper) IterateAuctionsByTime(ctx sdk.Context, inclusiveCutoffTime time.Time, cb func(auctionID uint64) (stop bool)) {
	store := k.byTimeStore(ctx)
	end := sdk.PrefixEndBytes(sdk.FormatTimeBytes(inclusiveCutoffTime)) // include inclusiveCutoffTime
	it := store.Iterator(nil, end)                                      // start at beginning of prefix store

	defer it.Close()
	for ; it.Valid(); it.Next() {
		auctionID := types.Uint64FromBytes(it.Value())
		if cb(auctionID) {
			break
		}
	}
}

// IterateAuctions provides an iterator over all stored auctions.
// For each auction, cb will be called. If cb returns true, iteration stops.
func (k Keeper) IterateAuctions(ctx sdk.Context, cb func(auction types.Auction) (stop bool)) {
	it := sdk.KVStorePrefixIterator(ctx.KVStore(k.storeKey), types.AuctionKeyPrefix)
	defer it.Close()

	for ; it.Valid(); it.Next() {
		auction := k.MustUnmarshalAuction(it.Value())
		if cb(auction) {
			break
		}
	}
}

// GetAllAuctions returns all auctions from the store.
func (k Keeper) GetAllAuctions(ctx sdk.Context) (auctions []types.Auction) {
	k.IterateAuctions(ctx, func(auction types.Auction) bool {
		auctions = append(auctions, auction)
		return false
	})
	return auctions
}
