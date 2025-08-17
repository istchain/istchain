# IstChain gRPC Client

The IstChain gRPC client is a tool for making gRPC queries on an IstChain chain.

## Features

- Easy-to-use gRPC client for the IstChain chain.
- Access all query clients for Cosmos and IstChain modules using `client.Query` (e.g., `client.Query.Bank.Balance`).
- Utilize utility functions for common queries (e.g., `client.BaseAccount(str)`).

## Usage

### Creating a new client

```go
package main

import (
  istchainGrpc "github.com/istchain/istchain/client/grpc"
)
grpcUrl := "https://grpc.istchain.io:443"
client, err := istchainGrpc.NewClient(grpcUrl)
if err != nil {
  panic(err)
}
```

### Making grpc queries

Query clients for both Cosmos and IstChain modules are available via `client.Query`.

Example: Query Cosmos module `x/bank` for address balance

```go
import (
  banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

rsp, err := client.Query.Bank.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: "ist19rjk5qmmwywnzfccwzyn02jywgpwjqf60afj92",
		Denom:   "uist",
	})
```

Example: Query IstChain module `x/evmutil` for params

```go
import (
  evmutiltypes "github.com/istchain/istchain/x/evmutil/types"
)

rsp, err := client.Query.Evmutil.Params(
  context.Background(), &evmutiltypes.QueryParamsRequest{},
)
```

#### Query Utilities

Utility functions for common queries are available directly on the client.

Example: Util query to get a base account

```go
istchainAcc := "ist19rjk5qmmwywnzfccwzyn02jywgpwjqf60afj92"
rsp, err := client.BaseAccount(istchainAcc)
if err != nil {
  panic(err)
}
fmt.Printf("account sequence for %s: %d\n", istchainAcc, rsp.Sequence)
```

## Query Tests

To test queries, an IstChain node is required. Therefore, the e2e tests for the gRPC client queries can be found in the `tests/e2e` directory. Tests for new utility queries should be added as e2e tests under the `test/e2e` directory.
