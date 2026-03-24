# IWS-Euler

模拟证券交易系统，采用微服务架构。链下高性能撮合 + 链上资产结算，参照 dYdX v3 设计思路。

## 架构总览

```
浏览器
  │  HTTP REST          WebSocket
  ▼                         ▼
WebApp (Nginx)
  │  /login /register /api/*    /ws
  ├─────────────────────────────────────► MarketData
  │                                         (K线聚合 + WS推送)
  ▼
Gateway (Gin)
  │  JWT鉴权 + Redis限流
  ▼
OrderService (Gin)
  │  写订单
  ▼
Kafka [orders topic]
  │
  ├──► MatchingEngine   ──► Kafka [trades topic]
  │     (撮合引擎)                     │
  │                           ├──► AccountService  (链下结算)
  │                           ├──► MarketData      (行情推送)
  │                           ├──► ChainClient     (链上结算)
  │                           └──► RiskControl     (风控检测)
  │
  └──► RiskControl (同时监听 orders + trades)

Anvil (本地链)
  └── Settlement 合约 (批量结算)
  └── MockERC20 合约 (测试 Token)
```

## 服务列表

| 服务                                                         | 语言            | 端口                | 职责                  |
| ------------------------------------------------------------ | --------------- | ------------------- | --------------------- |
| [WebApp](./WebApp/README.md)                         | React + Nginx   | 80 (NodePort 30080) | 前端 + 反向代理       |
| [Gateway](./Gateway/README.md)                       | Go (Gin)        | 8081                | 鉴权、限流、路由      |
| [OrderService](./OrderService/README.md)             | Go (Gin)        | 8082                | 订单接收、写 Kafka    |
| [MatchingEngine](./MatchingEngine/README.md)         | Go              | —                   | 订单撮合、发布成交    |
| [AccountService](./AccountService/README.md)         | Go              | —                   | 链下账户结算          |
| [MarketData](./MarketData/README.md)                 | Go              | 8080                | K线聚合、WS推送       |
| [ChainClient](./ChainClient/README.md)               | Go              | —                   | 调用合约批量结算      |
| [RiskControl](./RiskControl/README.md)               | Python          | —                   | 实时风控规则引擎      |
| [Anvil](./Anvil/README.md)                           | Shell (Foundry) | 8545                | 本地链 + 合约自动部署 |
| [SettlementContract](./SettlementContract/README.md) | Solidity        | —                   | 链上结算合约          |

## 快速开始

### 前提

- Docker Desktop（已启用 K8s）
- kubectl、helm、foundry（cast）

### 部署

```bash
# 1. 构建所有镜像
cd Anvil && docker build -t anvil:latest .
cd Gateway && docker build -t gateway:latest .
cd OrderService && docker build -t orderservice:latest .
cd MatchingEngine && docker build -t matchingengine:latest .
cd AccountService && docker build -t accountservice:latest .
cd MarketData && docker build -t marketdata:latest .
cd ChainClient && docker build -t chainclient:latest .
cd RiskControl && docker build -t riskcontrol:latest .
cd WebApp && docker build -t webapp:latest .

# 2. Helm 部署到 K8s
cd Deploy
helm upgrade --install iws-Euler . --namespace default

# 3. 访问 Web 界面
# http://localhost:30080
```

### 测试

```bash
# 单元测试（Go）
docker run --rm -v "$(pwd)/MatchingEngine:/workspace" wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v"

# 系统 E2E 测试
kubectl port-forward svc/gateway 18083:8081 -n iws-Euler
kubectl port-forward svc/marketdata 18084:8080 -n iws-Euler
python tests/e2e_system_test.py -v
```

详见 [docs/development.md](./docs/development.md)

## Kafka Topics

| Topic    | 生产者             | 消费者                                                               |
| -------- | ------------------ | -------------------------------------------------------------------- |
| `orders` | OrderService   | MatchingEngine、RiskControl                                  |
| `trades` | MatchingEngine | AccountService、MarketData、ChainClient、RiskControl |

## 文档导航

| 文档     | 路径                                              |
| -------- | ------------------------------------------------- |
| 系统架构 | [docs/architecture.md](./docs/architecture.md)    |
| 部署指南 | [docs/deployment.md](./docs/deployment.md)        |
| 开发指南 | [docs/development.md](./docs/development.md)      |
| 审计报告 | fix-summary-2026-03-22.md |
| 测试报告 | testing/test-report.md    |
