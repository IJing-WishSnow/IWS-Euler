package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/yourname/Gateway/handler"
	"github.com/yourname/Gateway/middleware"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	r := gin.Default()

	// CORS：开发阶段放开所有来源，允许 Authorization 头
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:    []string{"Content-Type", "Authorization"},
	}))

	// Redis 限流：每用户每秒最多 10 次请求
	limiter := middleware.NewRateLimiter(getEnv("REDIS_ADDR", "localhost:6379"), 10, time.Second)

	// 公开接口
	r.POST("/register", handler.Register)
	r.POST("/login", handler.Login)

	// 需要鉴权的接口
	auth := r.Group("/api", middleware.AuthRequired(), limiter.Limit())
	{
		auth.POST("/order", handler.PlaceOrder)
	}

	log.Println("Gateway 启动 :8081")
	r.Run(":8081")
}
