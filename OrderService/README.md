# OrderService

订单接收服务，负责参数校验、生成订单 ID 并写入 Kafka。

## 职责

- 接收来自 Gateway 转发的下单请求
- 参数校验：`quantity > 0`，`side` 必须为 `buy`/`sell`，`type` 必须为 `limit`/`market`
- 生成唯一 `order_id`（UUID）
- 将订单序列化后写入 Kafka `orders` topic

## 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/register` | 用户注册（转存 DB/内存） |
| POST | `/login` | 用户登录，返回 JWT |
| POST | `/order` | 接收下单请求（内部，由 Gateway 代理） |

## Kafka 消息格式（orders topic）

```json
{
  "order_id": "uuid-string",
  "user_id": "string",
  "symbol": "BTC/USDT",
  "side": "buy",
  "type": "limit",
  "price": 45000,
  "quantity": 2,
  "timestamp": 1711123456789
}
```

注意：`side` 和 `type` 字段序列化为小写字符串（自定义 JSON marshal），与 MatchingEngine 的解析逻辑兼容。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker 地址 |

## 目录结构

```
OrderService/
├── cmd/
│   └── main.go          # 启动入口
├── handler/
│   └── order.go         # 下单处理器，参数校验
├── service/
│   └── kafka.go         # Kafka Producer（仅 Producer，无 Consumer）
├── model/
│   └── order.go         # 订单数据结构
├── Dockerfile
└── go.mod
```

## 端口

- 容器端口：8082
- K8s Service：ClusterIP
