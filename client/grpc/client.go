package grpc

import (
	"errors"

	"github.com/istchain/istchain/client/grpc/query"
	"github.com/istchain/istchain/client/grpc/util"
)

// IstChainGrpcClient enables the usage of istchain grpc query clients and query utils
type IstChainGrpcClient struct {
	config IstChainGrpcClientConfig

	// Query clients for cosmos and istchain modules
	Query *query.QueryClient

	// Utils for common queries (ie fetch an unpacked BaseAccount)
	*util.Util
}

// IstChainGrpcClientConfig is a configuration struct for a IstChainGrpcClient
type IstChainGrpcClientConfig struct {
	// note: add future config options here
}

// NewClient creates a new IstChainGrpcClient via a grpc url
func NewClient(grpcUrl string) (*IstChainGrpcClient, error) {
	return NewClientWithConfig(grpcUrl, NewDefaultConfig())
}

// NewClientWithConfig creates a new IstChainGrpcClient via a grpc url and config
func NewClientWithConfig(grpcUrl string, config IstChainGrpcClientConfig) (*IstChainGrpcClient, error) {
	if grpcUrl == "" {
		return nil, errors.New("grpc url cannot be empty")
	}
	query, error := query.NewQueryClient(grpcUrl)
	if error != nil {
		return nil, error
	}
	client := &IstChainGrpcClient{
		Query:  query,
		Util:   util.NewUtil(query),
		config: config,
	}
	return client, nil
}

func NewDefaultConfig() IstChainGrpcClientConfig {
	return IstChainGrpcClientConfig{}
}
