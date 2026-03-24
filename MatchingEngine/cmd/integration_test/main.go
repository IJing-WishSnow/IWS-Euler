package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	KafkaBroker = "localhost:9094"
	OrdersTopic = "orders"
	TradesTopic = "trades"
)

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

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. 发送一对可以撮合的订单：卖单先进，买单后进
	// 卖单: alice 卖 10 BTC @ 50000
	// 买单: bob 买 10 BTC @ 50000
	// 价格相同，应该完全成交

	conn0, err := kafka.DialLeader(ctx, "tcp", KafkaBroker, OrdersTopic, 0)
	if err != nil {
		log.Fatalf("连接 Kafka 失败: %v", err)
	}
	defer conn0.Close()

	sellOrder := OrderMessage{
		ID: "sell-001", UserID: "alice", Symbol: "BTC/USDT",
		Side: "sell", Type: "limit", Price: 50000, Quantity: 10,
		CreatedAt: time.Now(),
	}
	buyOrder := OrderMessage{
		ID: "buy-001", UserID: "bob", Symbol: "BTC/USDT",
		Side: "buy", Type: "limit", Price: 50000, Quantity: 10,
		CreatedAt: time.Now().Add(time.Millisecond),
	}

	for _, o := range []OrderMessage{sellOrder, buyOrder} {
		data, _ := json.Marshal(o)
		conn0.SetWriteDeadline(time.Now().Add(3 * time.Second))
		if _, err := conn0.WriteMessages(kafka.Message{Value: data}); err != nil {
			log.Fatalf("发送订单失败 %s: %v", o.ID, err)
		}
		log.Printf("[TEST] 已发送订单: %s %s %s price=%d qty=%d",
			o.ID, o.Side, o.Symbol, o.Price, o.Quantity)
	}

	// 2. 等待撮合引擎处理，监听 trades topic 收成交回报
	log.Println("[TEST] 等待成交回报...")
	time.Sleep(2 * time.Second) // 等撮合引擎处理完

	// 根据 symbol key 计算分区：kafka-go hash 用 FNV
	// BTC/USDT 固定落在某个分区，直接遍历所有分区找
	for partition := 0; partition < 3; partition++ {
		tradeConn, err := kafka.DialLeader(ctx, "tcp", KafkaBroker, TradesTopic, partition)
		if err != nil {
			continue
		}

		lastOffset, _ := tradeConn.ReadLastOffset()
		firstOffset, _ := tradeConn.ReadFirstOffset()
		if lastOffset <= firstOffset {
			tradeConn.Close()
			continue
		}

		// 从最新的一条开始读
		tradeConn.Seek(lastOffset-1, kafka.SeekAbsolute)
		tradeConn.SetReadDeadline(time.Now().Add(3 * time.Second))
		batch := tradeConn.ReadBatch(1, 1e6)
		buf := make([]byte, 1e6)
		n, err := batch.Read(buf)
		batch.Close()
		tradeConn.Close()

		if err != nil || n == 0 {
			continue
		}

		var trade TradeMessage
		if err := json.Unmarshal(buf[:n], &trade); err != nil {
			continue
		}
		if trade.BuyOrderID != "buy-001" {
			continue // 不是本次测试的成交
		}

		fmt.Println("\n========== 全链路验证结果 ==========")
		fmt.Printf("成交ID:   %s\n", trade.ID)
		fmt.Printf("交易对:   %s\n", trade.Symbol)
		fmt.Printf("买方:     %s (订单 %s)\n", trade.BuyUserID, trade.BuyOrderID)
		fmt.Printf("卖方:     %s (订单 %s)\n", trade.SellUserID, trade.SellOrderID)
		fmt.Printf("成交价:   %d\n", trade.Price)
		fmt.Printf("成交量:   %d\n", trade.Quantity)
		fmt.Println("====================================\n")

		if trade.BuyUserID != "bob" || trade.SellUserID != "alice" {
			log.Fatal("全链路验证失败: 买卖方不符")
		}
		if trade.Price != 50000 || trade.Quantity != 10 {
			log.Fatal("全链路验证失败: 成交价或量不符")
		}

		fmt.Println("全链路验证通过: 下单 → Kafka → 撮合 → 成交回报")
		return
	}

	log.Fatal("全链路验证失败: 未找到本次成交回报，撮合引擎可能未运行")
}
