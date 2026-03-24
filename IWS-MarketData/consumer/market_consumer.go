package consumer

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/segmentio/kafka-go"
	"github.com/yourname/IWS-MarketData/kline"
	"github.com/yourname/IWS-MarketData/model"
	"github.com/yourname/IWS-MarketData/ws"
)

const TradesTopic = "trades"

var KafkaBroker = getEnv("KAFKA_BROKER", "localhost:9094")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// MarketConsumer 消费成交回报，聚合K线，推送WebSocket
type MarketConsumer struct {
	reader     *kafka.Reader
	aggregator *kline.Aggregator
	hub        *ws.Hub
}

func New(aggregator *kline.Aggregator, hub *ws.Hub) *MarketConsumer {
	return &MarketConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{KafkaBroker},
			Topic:       TradesTopic,
			GroupID:     "market-data",
			StartOffset: kafka.FirstOffset,
		}),
		aggregator: aggregator,
		hub:        hub,
	}
}

// Run 主循环
func (c *MarketConsumer) Run(ctx context.Context) error {
	log.Println("[MarketData] 行情消费者启动，等待成交...")

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
				log.Printf("[MarketData] 读取失败: %v", err)
				continue
			}

			var trade model.TradeMessage
			if err := json.Unmarshal(msg.Value, &trade); err != nil {
				log.Printf("[MarketData] 解析失败: %v", err)
				continue
			}

			// 1. 推送 tick
			tick := model.Tick{
				Symbol:   trade.Symbol,
				Price:    trade.Price,
				Quantity: trade.Quantity,
				Time:     trade.CreatedAt,
			}
			c.hub.BroadcastTick(tick)

			// 2. 喂入 K 线聚合器
			c.aggregator.Feed(trade.Symbol, trade.Price, trade.Quantity, trade.CreatedAt)

			// 3. 立即广播当前进行中的 K 线（不等周期结束）
			for _, interval := range []string{"1m", "5m", "1h"} {
				if current := c.aggregator.GetCurrent(trade.Symbol, interval); current != nil {
					c.hub.BroadcastKLine(*current)
				}
			}

			log.Printf("[MarketData] tick: %s price=%d qty=%d", trade.Symbol, trade.Price, trade.Quantity)
		}
	}
}

func (c *MarketConsumer) Close() {
	c.reader.Close()
}
