package consumer

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/segmentio/kafka-go"
	"github.com/yourname/ChainClient/client"
)

const (
	TradesTopic = "trades"
	// USDTDecimals USDT 精度：1 USDT = 10^6 units
	USDTDecimals = 1_000_000
	// BTCDecimals BTC 精度：1 BTC = 10^8 units
	BTCDecimals = 100_000_000
)

var KafkaBroker = getEnv("KAFKA_BROKER", "localhost:9094")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// TradeMessage 从 Kafka trades topic 消费的成交回报
type TradeMessage struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	BuyOrderID  string    `json:"buy_order_id"`
	BuyUserID   string    `json:"buy_user_id"`
	SellOrderID string    `json:"sell_order_id"`
	SellUserID  string    `json:"sell_user_id"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserAddressMap 用户名 → 以太坊地址映射（演示用，对应 Anvil 账户）
// 生产环境应从数据库读取用户注册的链上地址
var UserAddressMap = map[string]common.Address{
	"alice": common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"), // Anvil account 2
	"bob":   common.HexToAddress("0x90F79bf6EB2c4f870365E785982E1f101E93b906"), // Anvil account 3
}

// USDTAddress USDT 合约地址（Anvil nonce 0）
var USDTAddress = common.HexToAddress(getEnv("USDT_TOKEN_ADDR", "0x5FbDB2315678afecb367f032d93F642f64180aa3"))

// BTCAddress MockBTC 合约地址（Anvil nonce 2）
var BTCAddress = common.HexToAddress(getEnv("BTC_TOKEN_ADDR", "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"))

// TradeConsumer 消费成交回报并触发链上结算
type TradeConsumer struct {
	reader          *kafka.Reader
	settlementClient *client.SettlementClient
}

func New(settlementClient *client.SettlementClient) *TradeConsumer {
	return &TradeConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{KafkaBroker},
			Topic:       TradesTopic,
			GroupID:     "chain-client",
			StartOffset: kafka.LastOffset, // 只处理新成交，不重放历史
		}),
		settlementClient: settlementClient,
	}
}

// Run 主循环：消费 trades → 转换 → 调用链上 settle
func (c *TradeConsumer) Run(ctx context.Context) error {
	log.Println("[ChainClient] 链上结算消费者启动，等待成交...")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				log.Printf("[ChainClient] 读取 Kafka 失败: %v", err)
				continue
			}

			var trade TradeMessage
			if err := json.Unmarshal(msg.Value, &trade); err != nil {
				log.Printf("[ChainClient] 解析成交失败: %v", err)
				continue
			}

			log.Printf("[ChainClient] 收到成交: id=%s buy=%s sell=%s qty=%d price=%d",
				trade.ID, trade.BuyUserID, trade.SellUserID, trade.Quantity, trade.Price)

			settlements := c.buildSettlements(trade)
			if settlements == nil {
				log.Printf("[ChainClient] 跳过: 用户地址未找到 (buy=%s sell=%s)", trade.BuyUserID, trade.SellUserID)
				continue
			}

			if err := c.settlementClient.Settle(ctx, settlements); err != nil {
				log.Printf("[ChainClient] 链上结算失败: %v", err)
			}
		}
	}
}

// buildSettlements 将成交回报转换为链上结算指令（2条：USDT + BTC）
// BTC/USDT 成交：买方付 USDT → 卖方；卖方付 BTC → 买方
func (c *TradeConsumer) buildSettlements(trade TradeMessage) []client.Settlement {
	buyerAddr, buyerOK := UserAddressMap[strings.ToLower(trade.BuyUserID)]
	sellerAddr, sellerOK := UserAddressMap[strings.ToLower(trade.SellUserID)]
	if !buyerOK || !sellerOK {
		return nil
	}

	// USDT 金额：price（整数USDT）× quantity × 10^6
	usdtAmount := new(big.Int).Mul(
		big.NewInt(trade.Price*trade.Quantity),
		big.NewInt(USDTDecimals),
	)
	// BTC 金额：quantity × 10^8
	btcAmount := new(big.Int).Mul(
		big.NewInt(trade.Quantity),
		big.NewInt(BTCDecimals),
	)

	return []client.Settlement{
		{From: buyerAddr, To: sellerAddr, Token: USDTAddress, Amount: usdtAmount},  // 买方付 USDT
		{From: sellerAddr, To: buyerAddr, Token: BTCAddress, Amount: btcAmount},    // 卖方付 BTC
	}
}

func (c *TradeConsumer) Close() {
	c.reader.Close()
}
