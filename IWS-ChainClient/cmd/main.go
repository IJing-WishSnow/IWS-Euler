package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourname/IWS-ChainClient/client"
	"github.com/yourname/IWS-ChainClient/consumer"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rpcURL          := getEnv("ETHEREUM_RPC", "http://127.0.0.1:8545")
	contractAddress := getEnv("CONTRACT_ADDRESS", "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512")
	operatorPrivKey := getEnv("OPERATOR_PRIV_KEY", "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d")

	// 初始化链上客户端
	sc, err := client.NewSettlementClient(rpcURL, contractAddress, operatorPrivKey)
	if err != nil {
		log.Fatalf("初始化链上客户端失败: %v", err)
	}
	defer sc.Close()

	log.Println("IWS-ChainClient 启动")
	log.Printf("合约地址: %s", contractAddress)

	// 启动 Kafka 消费者
	tc := consumer.New(sc)
	defer tc.Close()

	if err := tc.Run(ctx); err != nil {
		log.Fatalf("链上结算消费者异常: %v", err)
	}

	log.Println("IWS-ChainClient 已停止")
}
