package consumer

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yourname/AccountService/service"
)

const TradesTopic = "trades"

var KafkaBroker = getEnv("KAFKA_BROKER", "localhost:9094")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// TradeMessage 成交回报（与 MatchingEngine 发出的格式一致）
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

// Settler 结算消费者：消费成交回报 → 调用 AccountService 结算
type Settler struct {
	account *service.AccountService
	reader  *kafka.Reader
}

func NewSettler(account *service.AccountService) *Settler {
	return &Settler{
		account: account,
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{KafkaBroker},
			Topic:       TradesTopic,
			GroupID:     "account-settler",
			StartOffset: kafka.FirstOffset,
		}),
	}
}

// Run 主循环（阻塞，应在 goroutine 中运行）
func (s *Settler) Run(ctx context.Context) error {
	log.Println("[Settler] 结算消费者启动，等待成交回报...")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := s.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				log.Printf("[Settler] 读取失败: %v", err)
				continue
			}

			var trade TradeMessage
			if err := json.Unmarshal(msg.Value, &trade); err != nil {
				log.Printf("[Settler] 解析失败: %v", err)
				continue
			}

			s.settle(trade)
		}
	}
}

// settle 结算一笔成交
// BTC/USDT 买入：买方扣冻结 USDT，加 BTC；卖方扣冻结 BTC，加 USDT
func (s *Settler) settle(trade TradeMessage) {
	parts := strings.Split(trade.Symbol, "/")
	if len(parts) != 2 {
		log.Printf("[Settler] 无效交易对: %s", trade.Symbol)
		return
	}
	baseAsset := parts[0]  // BTC
	quoteAsset := parts[1] // USDT

	quoteCost := trade.Price * trade.Quantity

	// 自动初始化新用户资金（模拟充值，开发阶段）
	s.account.EnsureAccount(trade.BuyUserID, quoteAsset, quoteCost*10)
	s.account.EnsureAccount(trade.SellUserID, baseAsset, trade.Quantity*10)

	// 买方结算：扣冻结的 USDT，加 BTC
	if err := s.account.SettleBuy(
		trade.BuyUserID, quoteAsset, quoteCost,
		baseAsset, trade.Quantity, trade.BuyOrderID,
	); err != nil {
		log.Printf("[Settler] 买方结算失败: %v (trade=%s)", err, trade.ID)
		return
	}

	// 卖方结算：扣冻结的 BTC，加 USDT
	if err := s.account.SettleSell(
		trade.SellUserID, baseAsset, trade.Quantity,
		quoteAsset, quoteCost, trade.SellOrderID,
	); err != nil {
		log.Printf("[Settler] 卖方结算失败: %v (trade=%s)", err, trade.ID)
		return
	}

	log.Printf("[Settler] 结算完成: %s 买方=%s 卖方=%s price=%d qty=%d",
		trade.Symbol, trade.BuyUserID, trade.SellUserID, trade.Price, trade.Quantity)
}

func (s *Settler) Close() {
	s.reader.Close()
}
