package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/yourname/IWS-MatchingEngine/bridge"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	b := bridge.New()
	defer b.Close()

	log.Println("IWS-MatchingEngine 启动")
	if err := b.Run(ctx); err != nil {
		log.Fatalf("撮合引擎异常退出: %v", err)
	}
	log.Println("IWS-MatchingEngine 已停止")
}
