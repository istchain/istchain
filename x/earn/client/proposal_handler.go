package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"github.com/istchain/istchain/x/earn/client/cli"
)

// community-pool deposit/withdraw proposal handlers
var (
	DepositProposalHandler  = govclient.NewProposalHandler(cli.GetCmdSubmitCommunityPoolDepositProposal)
	WithdrawProposalHandler = govclient.NewProposalHandler(cli.GetCmdSubmitCommunityPoolWithdrawProposal)
)
