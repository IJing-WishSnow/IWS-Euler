# ChainClient

链上结算客户端，消费成交消息，批量调用 IWSSettlement 合约完成链上资产结算。

## 职责

- 消费 Kafka `trades` topic
- 将成交记录转换为 `Settlement` 结构体（from, to, token, amount）
- 批量调用 IWSSettlement 合约的 `settle()` 函数
- 使用 Anvil 本地链（或其他 EVM 链）

## Kafka

| | Topic | 消费组 |
|--|-------|--------|
| 输入 | `trades` | `chainclient-group` |

## 合约交互

调用合约：`IWSSettlement.settle(Settlement[])`

```
合约地址：0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512（Anvil 确定性地址）
RPC：http://iws-anvil:8545
Operator 私钥：从环境变量 OPERATOR_PRIV_KEY 读取
```

Settlement 结构体（Solidity）：

```solidity
struct Settlement {
    address from;    // 卖方地址
    address to;      // 买方地址
    address token;   // ERC20 token 地址
    uint256 amount;  // 转移金额
}
```

## 环境变量

| 变量 | 说明 |
|------|------|
| `KAFKA_BROKERS` | Kafka broker 地址 |
| `RPC_URL` | 以太坊 RPC 端点（默认 `http://iws-anvil:8545`） |
| `CONTRACT_ADDR` | IWSSettlement 合约地址 |
| `OPERATOR_PRIV_KEY` | Operator 私钥（用于签名链上交易） |

## 目录结构

```
ChainClient/
├── cmd/
│   └── main.go                  # 启动入口
├── consumer/
│   └── trade_consumer.go        # Kafka 消费 + 聚合
├── client/
│   └── settlement.go            # 合约调用封装
├── Dockerfile
└── go.mod
```

## 无对外 HTTP 端口

ChainClient 是纯 Kafka 消费者，不暴露任何 HTTP 端口。
