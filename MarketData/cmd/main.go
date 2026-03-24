package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/yourname/MarketData/consumer"
	"github.com/yourname/MarketData/kline"
	"github.com/yourname/MarketData/model"
	"github.com/yourname/MarketData/ws"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// WebSocket Hub
	hub := ws.NewHub()

	// K线聚合器，K线完成时广播
	aggregator := kline.New(func(k model.KLine) {
		hub.BroadcastKLine(k)
	})

	// Kafka 消费者
	mc := consumer.New(aggregator, hub)
	defer mc.Close()

	// HTTP + WebSocket 服务
	http.HandleFunc("/ws", hub.HandleWS)
	go func() {
		log.Println("[MarketData] WebSocket 服务启动 :8080/ws")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("HTTP 服务失败: %v", err)
		}
	}()

	log.Println("MarketData 启动")
	if err := mc.Run(ctx); err != nil {
		log.Fatalf("行情服务异常退出: %v", err)
	}
	log.Println("MarketData 已停止")
}
