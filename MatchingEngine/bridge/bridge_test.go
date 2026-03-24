package bridge

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

// 测试 bridge 核心撮合逻辑（不依赖 Kafka 消费者组）
func TestBridgeProcessOrder(t *testing.T) {
	b := New()
	defer b.Close()

	// 先下买单，不会成交（没有对手方）
	buy := OrderMessage{
		ID: "buy-1", UserID: "alice", Symbol: "BTC/USDT",
		Side: "buy", Type: "limit", Price: 50000, Quantity: 10,
		CreatedAt: time.Now(),
	}
	trades := b.processOrder(buy)
	if len(trades) != 0 {
		t.Fatalf("买单不应产生成交，实际产生 %d 笔", len(trades))
	}

	// 再下卖单，价格匹配，应该成交
	sell := OrderMessage{
		ID: "sell-1", UserID: "bob", Symbol: "BTC/USDT",
		Side: "sell", Type: "limit", Price: 50000, Quantity: 10,
		CreatedAt: time.Now(),
	}
	trades = b.processOrder(sell)
	if len(trades) != 1 {
		t.Fatalf("期望 1 笔成交，实际 %d 笔", len(trades))
	}

	trade := trades[0]
	if trade.BuyUserID != "alice" || trade.SellUserID != "bob" {
		t.Fatalf("买卖方不对: buy=%s sell=%s", trade.BuyUserID, trade.SellUserID)
	}
	if trade.Price != 50000 || trade.Quantity != 10 {
		t.Fatalf("成交价或量不对: price=%d qty=%d", trade.Price, trade.Quantity)
	}

	t.Logf("撮合逻辑验证通过: buy=%s sell=%s price=%d qty=%d",
		trade.BuyUserID, trade.SellUserID, trade.Price, trade.Quantity)
}

// 集成测试：通过 Kafka 完整走一遍 订单→撮合→成交
func TestBridgeKafkaIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 启动 bridge
	b := New()
	defer b.Close()
	go b.Run(ctx)

	// 等 consumer group rebalance
	time.Sleep(5 * time.Second)

	// 写入买单和卖单
	conn, err := kafka.DialLeader(ctx, "tcp", KafkaBroker, OrdersTopic, 0)
	if err != nil {
		t.Fatalf("连接 Kafka 失败: %v", err)
	}

	buyData, _ := json.Marshal(OrderMessage{
		ID: "buy-k1", UserID: "alice", Symbol: "BTC/USDT",
		Side: "buy", Type: "limit", Price: 60000, Quantity: 5,
		CreatedAt: time.Now(),
	})
	sellData, _ := json.Marshal(OrderMessage{
		ID: "sell-k1", UserID: "bob", Symbol: "BTC/USDT",
		Side: "sell", Type: "limit", Price: 60000, Quantity: 5,
		CreatedAt: time.Now(),
	})

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.WriteMessages(
		kafka.Message{Value: buyData},
		kafka.Message{Value: sellData},
	)
	conn.Close()
	if err != nil {
		t.Fatalf("写入订单失败: %v", err)
	}
	t.Log("订单已写入 Kafka")

	// 等撮合完成
	time.Sleep(5 * time.Second)

	// 检查所有分区的 trades topic
	found := false
	for partition := 0; partition < 3; partition++ {
		tc, err := kafka.DialLeader(ctx, "tcp", KafkaBroker, TradesTopic, partition)
		if err != nil {
			continue
		}
		lastOffset, _ := tc.ReadLastOffset()
		if lastOffset == 0 {
			tc.Close()
			continue
		}
		// 读最后一条
		tc.Seek(lastOffset-1, kafka.SeekAbsolute)
		tc.SetReadDeadline(time.Now().Add(3 * time.Second))
		batch := tc.ReadBatch(1, 1e6)
		buf := make([]byte, 1e6)
		n, err := batch.Read(buf)
		batch.Close()
		tc.Close()
		if err != nil {
			continue
		}

		var trade TradeMessage
		if err := json.Unmarshal(buf[:n], &trade); err != nil {
			continue
		}
		if trade.BuyOrderID == "buy-k1" && trade.SellOrderID == "sell-k1" {
			found = true
			t.Logf("Kafka 全链路验证通过 (partition %d): price=%d qty=%d",
				partition, trade.Price, trade.Quantity)
			break
		}
	}

	if !found {
		t.Fatal("未在 trades topic 中找到成交回报")
	}
}
