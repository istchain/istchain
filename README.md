# IstChain

<p align="center">
  <img src="./istchain-logo.svg" width="300" alt="IstChain Logo">
</p>

<div align="center">

[![version](https://img.shields.io/github/tag/istchain/istchain.svg)](https://github.com/istchain/istchain/releases/latest)
[![CircleCI](https://circleci.com/gh/istchain/istchain/tree/master.svg?style=shield)](https://circleci.com/gh/istchain/istchain/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/istchain/istchain)](https://goreportcard.com/report/github.com/istchain/istchain)
[![API Reference](https://godoc.org/github.com/istchain/istchain?status.svg)](https://godoc.org/github.com/istchain/istchain)
[![GitHub](https://img.shields.io/github/license/istchain/istchain.svg)](https://github.com/istchain/istchain/blob/master/LICENSE.md)
[![Twitter Follow](https://img.shields.io/twitter/follow/ISTCHAIN.svg?label=Follow&style=social)](https://twitter.com/ISTCHAIN)
[![Discord Chat](https://img.shields.io/discord/704389840614981673.svg)](https://discord.com/invite/kQzh3Uv)

</div>

<div align="center">

### [Telegram](https://t.me/istchain) | [Medium](https://medium.com/@istchain) | [Discord](https://discord.gg/JJYnuCx)

</div>

## 概述

IstChain是一个基于Cosmos SDK构建的去中心化金融（DeFi）区块链平台，专注于跨链资产管理和金融创新。它提供了完整的DeFi基础设施，包括借贷、交易、流动性挖矿等功能。

## 主要特性

- **跨链DeFi**: 支持多链资产的无缝转移和交易
- **智能合约**: 基于Ethermint的EVM兼容性
- **流动性挖矿**: 创新的流动性激励机制
- **借贷平台**: 去中心化借贷和抵押系统
- **价格预言机**: 可靠的价格数据源
- **治理机制**: 去中心化社区治理
- **拍卖系统**: 去中心化资产拍卖
- **原子交换**: 跨链原子交换协议

## 快速开始

### 系统要求

- Go 1.21+
- Git
- Make

### 安装

```bash
# 克隆仓库
git clone https://github.com/istchain/istchain.git
cd istchain

# 安装依赖
make install

# 验证安装
istchaind version
```

### 运行本地节点

```bash
# 初始化节点
istchaind init mynode --chain-id istchain-local

# 添加测试账户
istchaind keys add mykey

# 启动节点
istchaind start
```

## 网络信息

### 主网

- **Chain ID**: istchain-1
- **推荐版本**: [v0.26.2](https://github.com/istchain/istchain/releases/tag/v0.26.2)
- **RPC端点**: https://rpc.istchain.io
- **API端点**: https://api.istchain.io
- **区块浏览器**: https://explorer.istchain.io

### 测试网

- **Chain ID**: istchain-testnet-1
- **测试网信息**: [测试网仓库](https://github.com/istchain/istchain-testnets)
- **测试网RPC**: https://testnet-rpc.istchain.io

## 开发指南

### 运行测试

```bash
# 运行单元测试
make test

# 运行集成测试
make test-e2e

# 运行模拟测试
make test-sim

# 运行所有测试
make test-all
```

### 构建

```bash
# 构建二进制文件
make build

# 构建Docker镜像
make docker-build

# 构建发布版本
make build-release
```

### 代码生成

```bash
# 生成Protocol Buffers代码
make proto-gen

# 生成API文档
make proto-docs

# 生成客户端代码
make proto-swagger-gen
```

## 核心模块

IstChain包含以下核心模块：

| 模块 | 功能描述 |
|------|----------|
| **auction** | 去中心化拍卖系统 |
| **bep3** | 跨链原子交换协议 |
| **cdp** | 抵押债务头寸管理 |
| **committee** | 委员会治理机制 |
| **community** | 社区池管理 |
| **earn** | 收益聚合器 |
| **evmutil** | EVM工具和转换 |
| **hard** | 去中心化借贷平台 |
| **incentive** | 流动性挖矿激励 |
| **issuance** | 代币发行管理 |
| **istdist** | 分发和奖励系统 |
| **liquid** | 流动性管理 |
| **pricefeed** | 价格预言机 |
| **router** | 跨模块路由 |
| **savings** | 储蓄账户 |
| **swap** | 去中心化交易 |

## API文档

- **REST API**: [API文档](https://docs.istchain.io/api)
- **gRPC**: [gRPC文档](https://docs.istchain.io/grpc)
- **EVM RPC**: [EVM RPC文档](https://docs.istchain.io/evm)
- **Swagger UI**: [Swagger文档](https://docs.istchain.io/swagger)

## 贡献指南

我们欢迎社区贡献！请查看我们的[贡献指南](CONTRIBUTING.md)。

### 开发环境设置

```bash
# 安装开发依赖
make install-dev

# 设置pre-commit钩子
make setup-hooks

# 运行代码检查
make lint

# 格式化代码
make format
```

### 提交代码

```bash
# 创建功能分支
git checkout -b feature/your-feature-name

# 提交更改
git commit -m "feat: add new feature"

# 推送分支
git push origin feature/your-feature-name
```

## 安全

如果您发现安全漏洞，请通过以下方式报告：

- **邮箱**: security@istchain.io
- **PGP密钥**: [安全密钥](https://istchain.io/security.asc)
- **漏洞赏金**: [Bug Bounty Program](https://istchain.io/bounty)

## 许可证

Copyright © IstChain Labs, Inc. All rights reserved.

本项目采用 [Apache v2 License](LICENSE.md) 许可证。

## 社区

- **官网**: https://istchain.io
- **文档**: https://docs.istchain.io
- **博客**: https://medium.com/@istchain
- **Twitter**: https://twitter.com/ISTCHAIN
- **Discord**: https://discord.gg/JJYnuCx
- **Telegram**: https://t.me/istchain
- **Reddit**: https://reddit.com/r/istchain

## 支持

如果您需要帮助或有技术问题，请：

1. 查看[文档](https://docs.istchain.io)
2. 在[Discord](https://discord.gg/JJYnuCx)中提问
3. 在[GitHub Issues](https://github.com/istchain/istchain/issues)中报告问题
4. 发送邮件到 support@istchain.io

## 路线图

- [ ] 跨链桥集成
- [ ] Layer 2 解决方案
- [ ] 移动端钱包
- [ ] 更多DeFi协议
- [ ] 治理代币发行

---

**注意**: `master`分支包含正在开发的功能，可能不稳定。生产环境请使用[发布版本](https://github.com/istchain/istchain/releases)。
