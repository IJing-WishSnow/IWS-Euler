package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/Gateway/middleware"
)

func getEnvHandler(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// 简单内存用户存储（后续接 PostgreSQL）
var (
	users   = map[string]string{} // username → password
	usersMu sync.RWMutex
)

var orderServiceClient = &http.Client{Timeout: 5 * time.Second}

// OrderRequest 下单请求（来自前端）
type OrderRequest struct {
	Symbol   string `json:"symbol" binding:"required"`
	Side     string `json:"side" binding:"required,oneof=buy sell"`
	Type     string `json:"type" binding:"required,oneof=limit market"`
	Price    int64  `json:"price"`
	Quantity int64  `json:"quantity" binding:"required,gt=0"`
}

// Register 注册
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	usersMu.Lock()
	_, exists := users[req.Username]
	if !exists {
		users[req.Username] = req.Password
	}
	usersMu.Unlock()
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "用户已存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "注册成功"})
}

// Login 登录，返回 JWT token
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	usersMu.RLock()
	pwd, exists := users[req.Username]
	usersMu.RUnlock()
	if !exists || pwd != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, err := middleware.GenerateToken(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 token 失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// PlaceOrder 下单（需要鉴权）—— 转发到 OrderService
func PlaceOrder(c *gin.Context) {
	userID := c.GetString("userID")

	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 构造转发给 OrderService 的请求体（注入 user_id）
	body := map[string]any{
		"user_id":  userID,
		"symbol":   req.Symbol,
		"side":     req.Side,
		"type":     req.Type,
		"price":    req.Price,
		"quantity": req.Quantity,
	}
	data, _ := json.Marshal(body)

	orderServiceURL := getEnvHandler("ORDER_SERVICE_URL", "http://localhost:8082")
	resp, err := orderServiceClient.Post(orderServiceURL+"/order", "application/json", bytes.NewReader(data))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "下单服务不可用"})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(respBody, &result)
	c.JSON(resp.StatusCode, result)
}
