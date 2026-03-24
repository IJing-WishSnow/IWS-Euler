# MatchingEngine

撮合引擎，消费订单消息，执行价格优先/时间优先撮合，发布成交结果。

## 职责

- 消费 Kafka `orders` topic
- 按 symbol 维护独立的订单簿（买单大根堆 + 卖单小根堆）
- 撮合规则：买价 >= 卖价时成交，价格优先，时间优先
- 将成交记录发布至 Kafka `trades` topic

## 订单簿数据结构

```
买单堆（最高价优先）  卖单堆（最低价优先）
  50100 buy 1          50000 sell 1
  49900 buy 2          50200 sell 3
  49800 buy 1          ...
```

撮合时：取买单堆顶（最高买价）和卖单堆顶（最低卖价），若买价 >= 卖价则成交。

## Kafka

|      | Topic    | 消费组                  |
| ---- | -------- | ----------------------- |
| 输入 | `orders` | `matching-engine-group` |
| 输出 | `trades` | —                       |

## 成交消息格式（trades topic）

```json
{
  "trade_id": "uuid-string",
  "symbol": "BTC/USDT",
  "buy_order_id": "...",
  "sell_order_id": "...",
  "buy_user_id": "alice",
  "sell_user_id": "bob",
  "price": 50000,
  "quantity": 1,
  "timestamp": 1711123456789
}
```

## 目录结构

```
MatchingEngine/
├── cmd/
│   ├── main.go                  # 启动入口
│   └── integration_test/
│       └── main.go              # Kafka 集成测试程序
├── engine/
│   ├── engine.go                # 订单簿 + 撮合核心逻辑
│   └── engine_test.go           # 撮合单元测试
├── bridge/
│   ├── bridge.go                # Kafka 消费/生产桥接层
│   └── bridge_test.go           # Kafka 集成测试
├── model/
│   └── order.go                 # 订单、成交数据结构
├── Dockerfile
└── go.mod
```

## 运行测试

```bash
# 撮合引擎单元测试
docker run --rm -v "T:/IWS-Euler/MatchingEngine:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./engine/... -v"

# Kafka 集成测试（需端口转发）
kubectl port-forward svc/kafka 9094:9092 -n iws-Euler
docker run --rm --network host \
  -v "T:/IWS-Euler/MatchingEngine:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./bridge/... -v"
```

## 无对外 HTTP 端口

MatchingEngine 是纯 Kafka 消费者/生产者，不暴露任何 HTTP 端口。
