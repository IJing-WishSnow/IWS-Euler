package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yourname/Gateway/middleware"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/register", Register)
	r.POST("/login", Login)
	auth := r.Group("/api", middleware.AuthRequired())
	{
		auth.POST("/order", PlaceOrder)
	}
	return r
}

func TestRegisterAndLogin(t *testing.T) {
	r := setupRouter()

	// 注册
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "123456"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("注册失败: %d %s", w.Code, w.Body.String())
	}
	t.Log("注册成功")

	// 登录
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("登录失败: %d %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	token := resp["token"]
	if token == "" {
		t.Fatal("未返回 token")
	}
	t.Logf("登录成功，token: %s...", token[:20])

	// 无 token 下单应被拒绝
	orderBody, _ := json.Marshal(map[string]any{
		"symbol": "BTC/USDT", "side": "buy", "type": "limit",
		"price": 50000, "quantity": 10,
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/order", bytes.NewBuffer(orderBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("无 token 应返回 401，实际 %d", w.Code)
	}
	t.Log("无 token 正确拒绝")

	// 带 token 下单（不连 Kafka，只验证鉴权通过）
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/order", bytes.NewBuffer(orderBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	// 如果 Kafka 没连上会返回 500，但鉴权已通过（不是 401）
	if w.Code == 401 {
		t.Fatal("带 token 不应返回 401")
	}
	t.Logf("鉴权通过，下单响应: %d %s", w.Code, w.Body.String())
}
