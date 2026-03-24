# 部署指南

## 前提条件

- Docker Desktop（已启用 Kubernetes）
- kubectl
- Helm 3
- Foundry（cast、anvil）— 可选，用于本地链交互

---

## 一、构建 Docker 镜像

所有镜像使用 `imagePullPolicy: Never`，必须在本地构建后才能在 K8s 中使用。

```bash
# 在项目根目录 T:/IWS-Euler/ 下执行

cd Anvil           && docker build -t iws-anvil:latest .           && cd ..
cd Gateway         && docker build -t iws-gateway:latest .         && cd ..
cd OrderService    && docker build -t iws-orderservice:latest .    && cd ..
cd MatchingEngine  && docker build -t iws-matchingengine:latest .  && cd ..
cd AccountService  && docker build -t iws-accountservice:latest .  && cd ..
cd MarketData      && docker build -t iws-marketdata:latest .      && cd ..
cd ChainClient     && docker build -t iws-chainclient:latest .     && cd ..
cd RiskControl     && docker build -t iws-riskcontrol:latest .     && cd ..
cd WebApp          && docker build -t iws-webapp:latest .          && cd ..
```

验证镜像列表：

```bash
docker images | grep iws-
```

---

## 二、Helm 部署

```bash
cd Deploy

# 首次部署
helm install iws-Euler . --namespace iws-Euler --create-namespace

# 更新部署（已存在时）
helm upgrade --install iws-Euler . --namespace iws-Euler
```

### 等待所有 Pod 就绪

```bash
kubectl get pods -n iws-Euler -w
```

预期所有 Pod 状态为 `Running`：

```
NAME                                  READY   STATUS    RESTARTS
iws-anvil-xxx                         1/1     Running   0
iws-gateway-xxx                       1/1     Running   0
iws-orderservice-xxx                  1/1     Running   0
iws-matchingengine-xxx                1/1     Running   0
iws-accountservice-xxx                1/1     Running   0
iws-marketdata-xxx                    1/1     Running   0
iws-chainclient-xxx                   1/1     Running   0
iws-riskcontrol-xxx                   1/1     Running   0
iws-webapp-xxx                        1/1     Running   0
kafka-xxx                             1/1     Running   0
redis-xxx                             1/1     Running   0
```

---

## 三、访问服务

### Web 界面

```
http://localhost:30080
```

### 端口转发（开发/测试用）

```bash
# Gateway API
kubectl port-forward svc/iws-gateway 18083:8081 -n iws-Euler

# MarketData WebSocket
kubectl port-forward svc/iws-marketdata 18084:8080 -n iws-Euler

# Redis（测试限流）
kubectl port-forward svc/redis 6379:6379 -n iws-Euler

# Kafka（集成测试）
kubectl port-forward svc/kafka 9094:9092 -n iws-Euler

# Anvil 本地链
kubectl port-forward svc/iws-anvil 8545:8545 -n iws-Euler
```

---

## 四、重新部署单个服务

当某个服务代码变更后，重新构建镜像并滚动重启：

```bash
# 以 Gateway 为例
cd Gateway && docker build -t iws-gateway:latest . && cd ..
kubectl rollout restart deployment/iws-gateway -n iws-Euler
kubectl rollout status deployment/iws-gateway -n iws-Euler
```

---

## 五、卸载

```bash
helm uninstall iws-Euler -n iws-Euler
kubectl delete namespace iws-Euler
```

---

## 六、配置说明

所有配置项集中在 `Deploy/values.yaml`：

| 字段                    | 说明                                 |
| ----------------------- | ------------------------------------ |
| `namespace`             | K8s 命名空间，默认 `iws-Euler`       |
| `kafka.image`           | Kafka 镜像                           |
| `redis.image`           | Redis 镜像                           |
| `anvil.contractAddress` | IWSSettlement 合约地址（确定性）     |
| `anvil.operatorPrivKey` | 后端调用合约的私钥（Anvil 测试账户） |
| `gateway.port`          | Gateway 监听端口（8081）             |
| `orderservice.port`     | OrderService 监听端口（8082）        |
| `marketdata.port`       | MarketData 监听端口（8080）          |

---

## 七、合约部署（Anvil）

Anvil 镜像启动时自动部署合约，无需手动操作。

合约地址（确定性，基于 Anvil 默认账户 nonce 0/1）：

| 合约          | 地址                                         |
| ------------- | -------------------------------------------- |
| MockERC20     | `0x5FbDB2315678afecb367f032d93F642f64180aa3` |
| IWSSettlement | `0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512` |

手动验证合约部署（需端口转发 Anvil）：

```bash
cast call 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512 \
  "owner()(address)" \
  --rpc-url http://localhost:8545
```

---

## 八、常见问题

### Pod CrashLoopBackOff

```bash
# 查看日志
kubectl logs -n iws-Euler deployment/iws-gateway --previous

# 常见原因：
# 1. Kafka 未就绪 → 等待 kafka pod Running 后其他服务会自动重试
# 2. 镜像未构建 → 执行 docker build 后重启
```

### 镜像找不到 (ErrImagePull)

所有镜像使用 `imagePullPolicy: Never`，必须在本地 Docker 中存在：

```bash
docker images | grep iws-
# 如果缺少某个，重新执行对应 docker build 命令
```

### 限流不生效

检查 Redis 连接和限流器配置：

```bash
kubectl port-forward svc/redis 6379:6379 -n iws-Euler
redis-cli ping  # 应返回 PONG
```
