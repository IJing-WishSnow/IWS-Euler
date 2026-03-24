# 系统架构文档

## 概述

IWS-Euler 是一个模拟证券交易系统，采用微服务架构，参照 dYdX v3 设计思路：**链下高性能撮合 + 链上资产结算**。

---

## 整体架构

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
  │  JWT鉴权 + Redis限流（滑动窗口 10次/秒）
  ▼
OrderService (Gin)
  │  写订单至 Kafka
  ▼
Kafka [orders topic]
  │
  ├──► MatchingEngine   ──► Kafka [trades topic]
  │     (撮合引擎)                     │
  │                           ├──► AccountService  (链下余额结算)
  │                           ├──► MarketData      (行情推送)
  │                           ├──► ChainClient     (链上合约结算)
  │                           └──► RiskControl     (风控检测)
  │
  └──► RiskControl (同时监听 orders + trades)

Anvil (本地链)
  └── Settlement 合约 (批量资产结算)
  └── MockERC20 合约 (测试 Token)
```

---

## 服务职责详解

### WebApp（前端 + 反向代理）

- React 单页应用，提供交易界面
- Nginx 反向代理：`/api/*` → Gateway，`/ws` → MarketData
- NodePort 30080 对外暴露，K8s 内部端口 80

### Gateway（网关）

- **JWT 鉴权**：所有 `/api/*` 接口要求 `Authorization: Bearer <token>`
- **Redis 限流**：基于 ZSET 滑动窗口，每用户每秒最多 10 次请求
- **路由转发**：下单请求代理至 OrderService（HTTP 反向代理）
- 公开接口：`POST /register`、`POST /login`
- 鉴权接口：`POST /api/order`
- 环境变量：`REDIS_ADDR`、`ORDER_SERVICE_URL`

### OrderService（订单服务）

- 接收网关转发的下单请求，进行参数校验
- 校验规则：`quantity > 0`，`side` 必须为 `buy`/`sell`，`type` 必须为 `limit`/`market`
- 生成唯一 `order_id`，写入 Kafka `orders` topic
- JSON 格式：side 字段序列化为 `"buy"`/`"sell"` 字符串

### MatchingEngine（撮合引擎）

- 消费 Kafka `orders` topic，按 symbol 维护独立订单簿
- 数据结构：买单大根堆（最高价优先），卖单小根堆（最低价优先）
- 撮合逻辑：买价 >= 卖价时成交，价格优先，时间优先
- 成交后发布至 Kafka `trades` topic
- 消费组：`matching-engine-group`

### AccountService（链下结算）

- 消费 Kafka `trades` topic，维护内存中的用户余额
- 更新买方/卖方的资产余额（token + USDT）
- 数据结构：`map[userID]map[token]float64`（使用 `sync.RWMutex` 保护）
- 消费组：`account-service-group`

### MarketData（行情服务）

- 消费 Kafka `trades` topic，聚合 K 线数据（按分钟）
- 通过 WebSocket 向浏览器实时推送成交行情
- Hub 广播机制：收集写失败的连接后统一清理（已修复 RWMutex 数据竞争）
- HTTP 端口 8080，WebSocket 路径 `/ws`
- 消费组：`marketdata-group`

### ChainClient（链上结算客户端）

- 消费 Kafka `trades` topic
- 聚合成交记录，批量调用 Settlement 合约的 `settle()` 函数
- 使用 Anvil 本地链（RPC: `http://anvil:8545`）
- 消费组：`chainclient-group`

### RiskControl（风控引擎）

- 同时消费 `orders` 和 `trades` 两个 topic
- 规则引擎（Python）：
  - **HIGH_FREQUENCY**：同一用户 60 秒内下单超过 20 次
  - **LARGE_ORDER**：单笔订单金额超过阈值（price × quantity）
  - **SPOOFING_SUSPECT**：同方向挂单数量异常（幌骗检测）
- 提供 HTTP 接口 `/summary` 查询风控统计
- 消费组：`riskcontrol-orders-group`、`riskcontrol-trades-group`

### Anvil（本地区块链）

- 基于 Foundry Anvil，启动本地以太坊节点
- 启动时自动部署 MockERC20 和 Settlement 合约
- RPC 端口 8545，链 ID 31337
- 合约地址（确定性）：
  - MockERC20: `0x5FbDB2315678afecb367f032d93F642f64180aa3`
  - Settlement: `0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512`

### SettlementContract（链上结算合约）

- `Settlement.sol`：核心合约
  - `deposit(token, amount)` — 用户充值 ERC20
  - `withdraw(token, amount)` — 用户提现
  - `settle(SettlementEntry[])` — operator 批量结算（转移链上余额）
  - `setOperator(addr)` — owner 更新 operator
- `MockERC20.sol`：无权限 mint 的测试用 ERC20
- 安全：ReentrancyGuard + SafeERC20 + Checks-Effects-Interactions

---

## 数据流

### 下单流程

```
用户 → WebApp → Gateway（JWT验证+限流）→ OrderService（参数校验）
    → Kafka[orders] → MatchingEngine（撮合）→ Kafka[trades]
    → AccountService（链下余额更新）
    → MarketData（行情推送 → WebSocket → 用户）
    → ChainClient（链上合约批量结算）
    → RiskControl（风控检测）
```

### 用户注册/登录流程

```
用户 → WebApp → Gateway → OrderService（/register /login）
    → 返回 JWT token → 存入浏览器本地存储
```

---

## Kafka Topics

| Topic    | 生产者             | 消费者                                                               |
| -------- | ------------------ | -------------------------------------------------------------------- |
| `orders` | OrderService   | MatchingEngine, RiskControl                                  |
| `trades` | MatchingEngine | AccountService, MarketData, ChainClient, RiskControl |

### 消费组列表

| 消费组                     | 服务           | Topic  |
| -------------------------- | -------------- | ------ |
| `matching-engine-group`    | MatchingEngine | orders |
| `account-service-group`    | AccountService | trades |
| `marketdata-group`         | MarketData     | trades |
| `chainclient-group`        | ChainClient    | trades |
| `riskcontrol-orders-group` | RiskControl    | orders |
| `riskcontrol-trades-group` | RiskControl    | trades |

---

## 基础设施

| 组件  | 镜像               | K8s 内部地址   | 用途      |
| ----- | ------------------ | -------------- | --------- |
| Kafka | apache/kafka:3.8.1 | kafka:9092     | 消息总线  |
| Redis | redis:7-alpine     | redis:6379     | 限流 ZSET |
| Anvil | anvil:latest   | anvil:8545 | 本地链    |

---

## 安全设计

| 层级   | 机制                                     |
| ------ | ---------------------------------------- |
| 接入层 | JWT 验证（HS256），Bearer Token          |
| 限流   | Redis ZSET 滑动窗口，每用户 10 次/秒     |
| 合约   | ReentrancyGuard，SafeERC20，onlyOperator |
| 并发   | sync.RWMutex（Go 服务），读写分离        |
| 消息   | Kafka consumer group，每服务独立消费位点 |

---

## K8s 部署拓扑

所有服务部署于 `iws-Euler` namespace，Helm chart 位于 `Deploy/`。

```
namespace: iws-Euler
  Deployments:
    webapp          (NodePort 30080)
    gateway         (ClusterIP :8081)
    orderservice    (ClusterIP :8082)
    matchingengine  (no port)
    accountservice  (no port)
    marketdata      (ClusterIP :8080)
    chainclient     (no port)
    riskcontrol     (no port)
    anvil           (ClusterIP :8545)
    kafka               (ClusterIP :9092)
    redis               (ClusterIP :6379)
```
