package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/IWS-OrderService/model"
	"github.com/yourname/IWS-OrderService/service"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// PlaceOrderRequest 来自 Gateway 的下单请求
type PlaceOrderRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Symbol   string `json:"symbol" binding:"required"`
	Side     string `json:"side" binding:"required,oneof=buy sell"`
	Type     string `json:"type" binding:"required,oneof=limit market"`
	Price    int64  `json:"price"`
	Quantity int64  `json:"quantity" binding:"required,gt=0"`
}

func main() {
	svc := service.NewOrderService()
	defer svc.Close()

	r := gin.Default()

	r.POST("/order", func(c *gin.Context) {
		var req PlaceOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var side model.Side
		if req.Side == "buy" {
			side = model.Buy
		} else {
			side = model.Sell
		}

		var orderType model.OrderType
		if req.Type == "limit" {
			orderType = model.Limit
		} else {
			orderType = model.Market
		}

		orderReq := model.OrderRequest{
			ID:        fmt.Sprintf("%s-%d", req.UserID, time.Now().UnixNano()),
			UserID:    req.UserID,
			Symbol:    req.Symbol,
			Side:      side,
			Type:      orderType,
			Price:     req.Price,
			Quantity:  req.Quantity,
			CreatedAt: time.Now(),
		}

		if err := svc.PlaceOrder(context.Background(), orderReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"order_id": orderReq.ID, "message": "下单成功"})
	})

	port := getEnv("PORT", "8082")
	log.Printf("IWS-OrderService 启动 :%s", port)
	r.Run(":" + port)
}
