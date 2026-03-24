# Deploy

Helm chart，将所有 IWS-Euler 服务部署到 Kubernetes。

## 结构

```
Deploy/
├── Chart.yaml              # Helm chart 元数据
├── values.yaml             # 所有服务的配置值
└── templates/
    ├── namespace.yaml      # 创建 iws-Euler namespace
    ├── kafka.yaml          # Kafka Deployment + Service
    ├── redis.yaml          # Redis Deployment + Service
    ├── anvil.yaml          # Anvil Deployment + Service
    ├── gateway.yaml        # Gateway Deployment + Service
    ├── orderservice.yaml   # OrderService Deployment + Service
    ├── matchingengine.yaml # MatchingEngine Deployment
    ├── accountservice.yaml # AccountService Deployment
    ├── marketdata.yaml     # MarketData Deployment + Service
    ├── chainclient.yaml    # ChainClient Deployment
    ├── riskcontrol.yaml    # RiskControl Deployment
    └── webapp.yaml         # WebApp Deployment + Service (NodePort)
```

## 部署

```bash
# 首次部署
helm install iws-Euler . --namespace iws-Euler --create-namespace

# 更新
helm upgrade --install iws-Euler . --namespace iws-Euler

# 卸载
helm uninstall iws-Euler -n iws-Euler
```

## 关键配置（values.yaml）

| 字段                    | 说明                                |
| ----------------------- | ----------------------------------- |
| `namespace`             | K8s 命名空间（`iws-Euler`）         |
| `anvil.contractAddress` | IWSSettlement 合约地址              |
| `anvil.operatorPrivKey` | Operator 私钥（Anvil 第二测试账户） |
| `gateway.port`          | 8081                                |
| `orderservice.port`     | 8082                                |
| `marketdata.port`       | 8080                                |
| `webapp.port`           | 80（NodePort 30080）                |

## 注意事项

- 所有业务服务使用 `imagePullPolicy: Never`，镜像必须本地构建
- Kafka 和 Redis 使用公共镜像，会从 Docker Hub 拉取
- 服务间通过 K8s Service DNS 名称通信（如 `kafka:9092`、`redis:6379`）
