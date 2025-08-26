package types

import (
	types "github.com/cosmos/cosmos-sdk/codec/types"
	v017auction "github.com/istchain/istchain/x/auction/types"
)

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterInterface(
		"istchain.auction.v1beta1.GenesisAuction",
		(*v017auction.GenesisAuction)(nil),
		&v017auction.SurplusAuction{},
		&v017auction.DebtAuction{},
		&v017auction.CollateralAuction{},
	)
}
