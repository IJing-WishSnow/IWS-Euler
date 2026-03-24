package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/yourname/AccountService/consumer"
	"github.com/yourname/AccountService/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	accountSvc := service.New()

	// 测试账户初始化：给 alice 和 bob 预存资金
	// alice: 100 BTC（用于卖出）
	// bob: 10000000 USDT（用于买入，50000 * 10 * 20 倍余量）
	accountSvc.Deposit("alice", "BTC", 100)
	accountSvc.Deposit("bob", "USDT", 10_000_000)
	accountSvc.Freeze("alice", "BTC", 10, "sell-001")  // 预冻结 alice 的卖单
	accountSvc.Freeze("bob", "USDT", 500_000, "buy-001") // 预冻结 bob 的买单（50000*10）
	log.Println("[AccountService] 测试账户已初始化: alice=100BTC, bob=10000000USDT")

	settler := consumer.NewSettler(accountSvc)
	defer settler.Close()

	log.Println("AccountService 启动")
	if err := settler.Run(ctx); err != nil {
		log.Fatalf("账户服务异常退出: %v", err)
	}
	log.Println("AccountService 已停止")
}
