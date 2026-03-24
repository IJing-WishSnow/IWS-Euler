# IWS-Euler — Claude 上下文

## 项目简介

模拟证券交易系统，参照 dYdX v3 设计：**链下高性能撮合 + 链上资产结算**。
部署目标：本地 Docker Desktop Kubernetes。

---

## 目录结构

```
IWS-Euler/
├── Gateway/            Go (Gin)      JWT鉴权 + Redis限流 + 路由转发  :8081
├── OrderService/       Go (Gin)      订单接收 → Kafka [orders]       :8082
├── MatchingEngine/     Go            撮合引擎 → Kafka [trades]
├── AccountService/     Go            链下余额结算（消费 trades）
├── MarketData/         Go            K线聚合 + WebSocket推送          :8080
├── ChainClient/        Go            调用链上合约批量结算
├── RiskControl/        Python 3.11   实时风控规则引擎
├── WebApp/             React + Nginx 前端 + 反向代理                  :30080
├── Anvil/              Foundry/Shell 本地链 + 合约自动部署            :8545
├── SettlementContract/ Solidity      链上结算合约（Hardhat）
├── Infra/              docker-compose 本地依赖（Kafka/Redis/PG）
├── Deploy/             Helm chart    K8s 部署（namespace: iws-Euler）
├── docs/               架构/部署/开发文档
├── testing/            测试计划 + 报告
└── tests/              e2e_system_test.py 系统端到端测试
```

## Go 模块命名

各 Go 服务模块名格式：`github.com/yourname/<ServiceName>`
（如 `github.com/yourname/Gateway`、`github.com/yourname/MatchingEngine`）

---

## 常用命令

### 本地依赖（Kafka / Redis / PostgreSQL）

```bash
cd Infra
docker compose up -d        # 启动
docker compose down         # 停止
```

端口：Kafka `9094`（外部）/ `9092`（内部）、Redis `6379`、PostgreSQL `5432`

### 运行 Go 单元测试

不需要本地 Go 环境，用 `wishsnow/golang-dev:latest` 镜像：

```bash
# 示例：MatchingEngine
docker run --rm -v "T:/IWS-Euler/MatchingEngine:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v"

# 其他服务同理，替换路径和目录名即可
```

### Gateway 限流集成测试（需要 Redis）

```bash
kubectl port-forward svc/redis 6379:6379 -n iws-Euler &
docker run --rm --network host -v "T:/IWS-Euler/Gateway:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v -run Integration"
```

### RiskControl 单元测试

```bash
cd RiskControl
python -m pytest tests/ -v
# 或用 docker compose 跑隔离环境
```

### 系统 E2E 测试

```bash
# 需要系统已通过 Helm 部署且所有 Pod Running
kubectl port-forward svc/gateway 8081:8081 -n iws-Euler &
kubectl port-forward svc/marketdata 8080:8080 -n iws-Euler &
python tests/e2e_system_test.py
```

### 构建所有 Docker 镜像

```bash
for svc in Anvil Gateway OrderService MatchingEngine AccountService MarketData ChainClient RiskControl WebApp; do
  cd $svc && docker build -t ${svc,,}:latest . && cd ..
done
```

### Helm 部署 / 更新

```bash
cd Deploy
helm upgrade --install iws-Euler . --namespace iws-Euler --create-namespace
kubectl get pods -n iws-Euler -w
```

### 查看服务日志

```bash
kubectl logs -n iws-Euler -l app=gateway --tail=50 -f
kubectl logs -n iws-Euler -l app=matchingengine --tail=50 -f
```

---

## 关键架构约定

- **消息总线**：Kafka topic `orders`（下单）、`trades`（成交）
- **限流**：Gateway 用 Redis ZSET 滑动窗口，10次/秒/用户
- **认证**：JWT，所有 `/api/*` 接口需要 `Authorization: Bearer <token>`
- **K8s namespace**：`iws-Euler`
- **镜像策略**：`imagePullPolicy: Never`，必须本地 build
- **链上集群 ID**：`IWS-Euler-kafka-cluster`（此处保留 IWS-Euler 前缀）
- **本地链**：Anvil 监听 `:8545`，合约地址写在 `Deploy/values.yaml` 的 `anvil.contractAddress`

---

## 不在测试范围内

- Anvil 链上合约功能测试（需链上环境）
- ChainClient 链上结算（无预存余额）
- 性能压测

---

## 技术栈速查

| 层 | 技术 |
|---|---|
| Go 服务 | Go 1.22+, Gin, kafka-go, go-redis, golang-jwt |
| Python 服务 | Python 3.11+, kafka-python, openai |
| 前端 | React, Nginx |
| 合约 | Solidity, Hardhat, Foundry |
| 消息 | Apache Kafka 3.8 (KRaft 模式，无 ZooKeeper) |
| 存储 | PostgreSQL 16, Redis 7 |
| 容器编排 | Docker Desktop K8s, Helm 3 |
| CI 测试镜像 | `wishsnow/golang-dev:latest` |
