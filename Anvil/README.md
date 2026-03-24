# Anvil

本地以太坊节点，基于 Foundry Anvil，启动时自动部署 MockERC20 和 Settlement 合约。

## 职责

- 运行本地以太坊节点（链 ID 31337）
- 启动脚本自动部署两个合约：
  1. `MockERC20`：无权限 mint 的测试用 ERC20
  2. `Settlement`：资产托管和批量结算合约

## 合约地址（确定性）

基于 Anvil 默认账户（`0xf39Fd6e51...`）的 nonce 推导：

| 合约          | 地址                                         |
| ------------- | -------------------------------------------- |
| MockERC20     | `0x5FbDB2315678afecb367f032d93F642f64180aa3` |
| Settlement | `0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512` |

> 这些地址已硬编码在 `Deploy/values.yaml` 的 `anvil.contractAddress` 中。

## Anvil 配置

- RPC 端口：8545
- 链 ID：31337
- 预置账户：10 个测试账户，每个 10000 ETH
- 默认部署账户：`0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266`
- Operator（合约第二账户）：`0x70997970C51812dc3A010C7d01b50e0d17dc79C8`

## 目录结构

```
Anvil/
├── deploy.sh      # 启动 Anvil + 部署合约脚本
├── Dockerfile     # 基于 ghcr.io/foundry-rs/foundry
└── README.md
```

## 手动交互（需端口转发）

```bash
kubectl port-forward svc/anvil 8545:8545 -n iws-Euler

# 查询合约 owner
cast call 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512 \
  "owner()(address)" --rpc-url http://localhost:8545

# Mint 测试 token
cast send 0x5FbDB2315678afecb367f032d93F642f64180aa3 \
  "mint(address,uint256)" \
  0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 1000000000000000000 \
  --rpc-url http://localhost:8545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

## 端口

- 容器端口：8545
- K8s Service：ClusterIP（服务名 `anvil`）
