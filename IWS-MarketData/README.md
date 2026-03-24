# IWS-MarketData

行情服务，消费成交数据，聚合 K 线，通过 WebSocket 实时推送给浏览器。

## 职责

- 消费 Kafka `trades` topic
- 聚合分钟级 K 线（OHLCV）
- 通过 WebSocket 广播实时行情
- Hub 模式管理所有 WebSocket 连接

## WebSocket 协议

连接地址：`ws://localhost:18084/ws`（端口转发后）

推送消息格式：

```json
{
  "type": "trade",
  "symbol": "BTC/USDT",
  "price": 50000,
  "quantity": 1,
  "timestamp": 1711123456789
}
```

K 线消息格式：

```json
{
  "type": "kline",
  "symbol": "BTC/USDT",
  "open": 49900,
  "high": 50100,
  "low": 49800,
  "close": 50000,
  "volume": 10,
  "ts": 1711123440000
}
```

## Kafka

| | Topic | 消费组 |
|--|-------|--------|
| 输入 | `trades` | `marketdata-group` |

## 目录结构

```
IWS-MarketData/
├── cmd/
│   └── main.go       # 启动入口
├── ws/
│   └── hub.go        # WebSocket Hub，广播 + 连接生命周期管理
├── kline/
│   └── aggregator.go # K 线聚合逻辑
├── consumer/
│   └── consumer.go   # Kafka 消费者
├── Dockerfile
└── go.mod
```

## 并发安全说明

Hub 广播采用两阶段策略（已修复数据竞争问题）：

1. 持有 `RLock` 遍历所有连接，尝试写消息，收集写失败的连接到 `dead` 列表
2. 释放 `RLock`，持有 `Lock`，批量关闭并从 map 中删除失效连接

不可在 `RLock` 期间执行 `delete(map)`，否则触发 data race。

## 端口

- 容器端口：8080
- K8s Service：ClusterIP
- 路径：`/ws`（WebSocket）
