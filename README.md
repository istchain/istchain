# IstChain

<p align="center">
  <img src="./istchain-logo.svg" width="300" alt="IstChain Logo">
</p>

<div align="center">

[![GitHub](https://img.shields.io/github/license/istchain/istchain.svg)](https://github.com/istchain/istchain/blob/master/LICENSE.md)
[![Twitter Follow](https://img.shields.io/twitter/follow/ISTCHAIN.svg?label=Follow&style=social)](https://twitter.com/ISTCHAIN)

</div>

<div align="center">

</div>

## Overview
IstChain is a decentralized finance (DeFi) blockchain platform built on the Cosmos SDK, focusing on cross-chain asset management and financial innovation. It provides a comprehensive DeFi infrastructure, including lending, trading, liquidity mining, and more.

## Key Features
- **Cross-Chain DeFi**: Supports seamless transfer and trading of multi-chain assets
- **Smart Contracts**: EVM compatibility based on Ethermint
- **Liquidity Mining**: Innovative liquidity incentive mechanisms
- **Lending Platform**: Decentralized lending and collateral system
- **Price Oracle**: Reliable price data sources
- **Governance Mechanism**: Decentralized community governance
- **Auction System**: Decentralized asset auctions
- **Atomic Swaps**: Cross-chain atomic swap protocol

## Quick Start

### System Requirements

- Go 1.21+
- Git
- Make

### Installation

```bash
# Clone the repository  
git clone https://github.com/istchain/istchain.git
cd istchain

# Install dependencies  
make install

# Verify installation
istchaind version
```

### Running a Local Node

```bash
# Initialize the node  
istchaind init mynode --chain-id istchain-local

# Add a test account  
istchaind keys add mykey

# Start the node  
istchaind start
```

## Development Guide

### Running Tests

```bash
# Run unit tests
make test

# Run integration tests
make test-e2e

# Run simulation tests
make test-sim

# Run all tests
make test-all
```

### Building

```bash
# Build binary files
make build

# Build Docker image
make docker-build

# Build release version
make build-release
```

### Code Generation

```bash
# Generate Protocol Buffers code
make proto-gen

# Generate API documentation
make proto-docs

# Generate client code
make proto-swagger-gen
```

## Core Modules

IstChain includes the following core modules:

| Module | Description |
|------|----------|
| **auction** | Decentralized auction system |
| **bep3** | Cross-chain atomic swap protocol |
| **cdp** | Collateralized debt position management |
| **committee** | Committee governance mechanism |
| **community** | Community pool management |
| **earn** | Yield aggregator |
| **evmutil** | EVM utilities and conversions |
| **hard** | Decentralized lending platform |
| **incentive** | Liquidity mining incentives |
| **issuance** | Token issuance management |
| **istdist** | Distribution and reward system |
| **liquid** | Liquidity management |
| **pricefeed** | Price oracle |
| **router** | Cross-module routing |
| **savings** | Savings accounts |
| **swap** | Decentralized exchange |

## Contribution Guidelines
We welcome community contributions! Please check our [Contribution Guidelines](CONTRIBUTING.md).

### Development Environment Setup

```bash
# Install development dependencies  
make install-dev  

# Set up pre-commit hooks  
make setup-hooks  

# Run code checks  
make lint  

# Format code  
make format  
```

### Submitting Code

```bash
# Create a feature branch
git checkout -b feature/your-feature-name

# Commit changes  
git commit -m "feat: add new feature"

# Push the branch  
git push origin feature/your-feature-name
```

## License

Copyright © IstChain Labs, Inc. All rights reserved.

This project is licensed under the [Apache v2 License](LICENSE.md) 

## Community

- **Website**: [http://istchain.us](http://istchain.us/)
- **Twitter**: [https://x.com/IST_off](https://x.com/IST_off)

---
Note: The master branch contains features under development and may be unstable. For production environments, please use the released versions.


