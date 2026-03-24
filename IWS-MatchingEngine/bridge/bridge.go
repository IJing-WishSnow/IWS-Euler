package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yourname/IWS-MatchingEngine/engine"
	"github.com/yourname/IWS-MatchingEngine/model"
)

const (
	OrdersTopic = "orders"
	TradesTopic = "trades"
)

var KafkaBroker = getEnv("KAFKA_BROKER", "localhost:9094")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// OrderMessage Kafka 中的订单消息格式（与 OrderService 一致）
type OrderMessage struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Type      string    `json:"type"`
	Price     int64     `json:"price"`
	Quantity  int64     `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

// TradeMessage 成交回报消息格式（发到 Kafka，AccountService 消费）
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

// Bridge 撮合引擎的 Kafka 桥接层
type Bridge struct {
	engines map[string]*engine.Engine // key: symbol
	users   map[string]string        // key: orderID → userID（追踪订单归属）
	reader  *kafka.Reader
	writer  *kafka.Writer
}

func New() *Bridge {
	return &Bridge{
		engines: make(map[string]*engine.Engine),
		users:   make(map[string]string),
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{KafkaBroker},
			Topic:       OrdersTopic,
			GroupID:     "matching-engine",
			StartOffset: kafka.FirstOffset,
		}),
		writer: &kafka.Writer{
			Addr:     kafka.TCP(KafkaBroker),
			Topic:    TradesTopic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

// Run 主循环：消费订单 → 撮合 → 发布成交（阻塞，应在 goroutine 中运行）
func (b *Bridge) Run(ctx context.Context) error {
	log.Println("[Bridge] 撮合引擎桥接启动，等待订单...")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := b.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				log.Printf("[Bridge] 读取订单失败: %v", err)
				continue
			}

			var om OrderMessage
			if err := json.Unmarshal(msg.Value, &om); err != nil {
				log.Printf("[Bridge] 解析订单失败: %v", err)
				continue
			}
			log.Printf("[Bridge] 收到订单: id=%s user=%s side=%s price=%d qty=%d", om.ID, om.UserID, om.Side, om.Price, om.Quantity)

			trades := b.processOrder(om)

			for _, t := range trades {
				if err := b.publishTrade(ctx, t); err != nil {
					log.Printf("[Bridge] 发布成交失败: %v", err)
				}
			}
		}
	}
}

// processOrder 处理单个订单：转换格式 → 撮合 → 转换成交回报
func (b *Bridge) processOrder(om OrderMessage) []TradeMessage {
	// 获取或创建该交易对的引擎
	eng, ok := b.engines[om.Symbol]
	if !ok {
		eng = engine.New(om.Symbol)
		b.engines[om.Symbol] = eng
	}

	// 记录订单归属
	b.users[om.ID] = om.UserID

	// 转换为引擎内部格式
	order := &model.Order{
		ID:        om.ID,
		UserID:    om.UserID,
		Symbol:    om.Symbol,
		Side:      parseSide(om.Side),
		Type:      parseType(om.Type),
		Price:     om.Price,
		Quantity:  om.Quantity,
		CreatedAt: om.CreatedAt,
	}

	// 撮合
	trades := eng.PlaceOrder(order)

	// 转换成交回报
	// 注意：不在此处删除 b.users 中的条目，因为订单支持部分成交（PartiallyFilled），
	// 同一 orderID 可能对应多笔成交，提前删除会导致后续成交的 UserID 为空。
	var msgs []TradeMessage
	for _, t := range trades {
		msgs = append(msgs, TradeMessage{
			ID:          t.ID,
			Symbol:      t.Symbol,
			BuyOrderID:  t.BuyOrderID,
			BuyUserID:   b.users[t.BuyOrderID],
			SellOrderID: t.SellOrderID,
			SellUserID:  b.users[t.SellOrderID],
			Price:       t.Price,
			Quantity:    t.Quantity,
			CreatedAt:   t.CreatedAt,
		})
	}

	if len(msgs) > 0 {
		log.Printf("[Bridge] 撮合完成: symbol=%s 产生 %d 笔成交", om.Symbol, len(msgs))
	}

	return msgs
}

func (b *Bridge) publishTrade(ctx context.Context, t TradeMessage) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("序列化成交回报失败: %w", err)
	}
	return b.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(t.Symbol),
		Value: data,
	})
}

func (b *Bridge) Close() {
	b.reader.Close()
	b.writer.Close()
}

func parseSide(s string) model.Side {
	if s == "sell" {
		return model.Sell
	}
	return model.Buy
}

func parseType(t string) model.OrderType {
	if t == "market" {
		return model.Market
	}
	return model.Limit
}
