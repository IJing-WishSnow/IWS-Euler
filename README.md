# IWS-Euler

模拟证券交易系统，采用微服务架构。链下高性能撮合 + 链上资产结算，参照 dYdX v3 设计思路。

## 架构总览

```
浏览器
  │  HTTP REST          WebSocket
  ▼                         ▼
IWS-WebApp (Nginx)
  │  /login /register /api/*    /ws
  ├─────────────────────────────────────► IWS-MarketData
  │                                         (K线聚合 + WS推送)
  ▼
IWS-Gateway (Gin)
  │  JWT鉴权 + Redis限流
  ▼
IWS-OrderService (Gin)
  │  写订单
  ▼
Kafka [orders topic]
  │
  ├──► IWS-MatchingEngine   ──► Kafka [trades topic]
  │     (撮合引擎)                     │
  │                           ├──► IWS-AccountService  (链下结算)
  │                           ├──► IWS-MarketData      (行情推送)
  │                           ├──► IWS-ChainClient     (链上结算)
  │                           └──► IWS-RiskControl     (风控检测)
  │
  └──► IWS-RiskControl (同时监听 orders + trades)

IWS-Anvil (本地链)
  └── IWSSettlement 合约 (批量结算)
  └── MockERC20 合约 (测试 Token)
```

## 服务列表

| 服务                                                         | 语言            | 端口                | 职责                  |
| ------------------------------------------------------------ | --------------- | ------------------- | --------------------- |
| [IWS-WebApp](./IWS-WebApp/README.md)                         | React + Nginx   | 80 (NodePort 30080) | 前端 + 反向代理       |
| [IWS-Gateway](./IWS-Gateway/README.md)                       | Go (Gin)        | 8081                | 鉴权、限流、路由      |
| [IWS-OrderService](./IWS-OrderService/README.md)             | Go (Gin)        | 8082                | 订单接收、写 Kafka    |
| [IWS-MatchingEngine](./IWS-MatchingEngine/README.md)         | Go              | —                   | 订单撮合、发布成交    |
| [IWS-AccountService](./IWS-AccountService/README.md)         | Go              | —                   | 链下账户结算          |
| [IWS-MarketData](./IWS-MarketData/README.md)                 | Go              | 8080                | K线聚合、WS推送       |
| [IWS-ChainClient](./IWS-ChainClient/README.md)               | Go              | —                   | 调用合约批量结算      |
| [IWS-RiskControl](./IWS-RiskControl/README.md)               | Python          | —                   | 实时风控规则引擎      |
| [IWS-Anvil](./IWS-Anvil/README.md)                           | Shell (Foundry) | 8545                | 本地链 + 合约自动部署 |
| [IWS-SettlementContract](./IWS-SettlementContract/README.md) | Solidity        | —                   | 链上结算合约          |

## 快速开始

### 前提

- Docker Desktop（已启用 K8s）
- kubectl、helm、foundry（cast）

### 部署

```bash
# 1. 构建所有镜像
cd IWS-Anvil && docker build -t iws-anvil:latest .
cd IWS-Gateway && docker build -t iws-gateway:latest .
cd IWS-OrderService && docker build -t iws-orderservice:latest .
cd IWS-MatchingEngine && docker build -t iws-matchingengine:latest .
cd IWS-AccountService && docker build -t iws-accountservice:latest .
cd IWS-MarketData && docker build -t iws-marketdata:latest .
cd IWS-ChainClient && docker build -t iws-chainclient:latest .
cd IWS-RiskControl && docker build -t iws-riskcontrol:latest .
cd IWS-WebApp && docker build -t iws-webapp:latest .

# 2. Helm 部署到 K8s
cd IWS-Deploy
helm upgrade --install iws-Euler . --namespace default

# 3. 访问 Web 界面
# http://localhost:30080
```

### 测试

```bash
# 单元测试（Go）
docker run --rm -v "$(pwd)/IWS-MatchingEngine:/workspace" wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v"

# 系统 E2E 测试
kubectl port-forward svc/iws-gateway 18083:8081 -n iws-Euler
kubectl port-forward svc/iws-marketdata 18084:8080 -n iws-Euler
python tests/e2e_system_test.py -v
```

详见 [docs/development.md](./docs/development.md)

## Kafka Topics

| Topic    | 生产者             | 消费者                                                               |
| -------- | ------------------ | -------------------------------------------------------------------- |
| `orders` | IWS-OrderService   | IWS-MatchingEngine、IWS-RiskControl                                  |
| `trades` | IWS-MatchingEngine | IWS-AccountService、IWS-MarketData、IWS-ChainClient、IWS-RiskControl |

## 文档导航

| 文档     | 路径                                              |
| -------- | ------------------------------------------------- |
| 系统架构 | [docs/architecture.md](./docs/architecture.md)    |
| 部署指南 | [docs/deployment.md](./docs/deployment.md)        |
| 开发指南 | [docs/development.md](./docs/development.md)      |
| 审计报告 | fix-summary-2026-03-22.md |
| 测试报告 | testing/test-report.md    |
