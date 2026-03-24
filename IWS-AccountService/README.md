# IWS-AccountService

链下账户结算服务，消费成交消息，更新内存中的用户余额。

## 职责

- 消费 Kafka `trades` topic
- 根据每笔成交记录更新买方和卖方的链下余额
- 买方：扣减 USDT，增加 token（symbol 中的基础资产）
- 卖方：扣减 token，增加 USDT
- 使用 `sync.RWMutex` 保护并发访问

## 余额数据结构

```
balances map[userID]map[asset]float64

示例：
  "alice" → {"BTC": 1.0, "USDT": -50000.0}
  "bob"   → {"BTC": -1.0, "USDT": 50000.0}
```

## Kafka

|      | Topic    | 消费组                  |
| ---- | -------- | ----------------------- |
| 输入 | `trades` | `account-service-group` |

## 目录结构

```
IWS-AccountService/
├── cmd/
│   └── main.go                      # 启动入口
├── consumer/
│   └── settler.go                   # Kafka 消费 + 余额更新
├── service/
│   ├── account_service.go           # 余额管理逻辑
│   └── account_service_test.go      # 单元测试
├── model/
│   └── account.go                   # 账户数据结构
├── Dockerfile
└── go.mod
```

## 运行测试

```bash
docker run --rm -v "T:/IWS-Euler/IWS-AccountService:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v"
```

## 注意事项

- 当前余额存储在内存中，服务重启后数据丢失（适合 demo 场景）
- 链下余额与链上 IWSSettlement 合约余额相互独立
- `sync.RWMutex`：读操作使用 `RLock`，写操作使用 `Lock`，不可在 `RLock` 内修改 map

## 无对外 HTTP 端口

AccountService 是纯 Kafka 消费者，不暴露任何 HTTP 端口。
