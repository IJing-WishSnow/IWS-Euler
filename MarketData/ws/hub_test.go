package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourname/MarketData/model"
)

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()

	// 启动测试 HTTP 服务
	server := httptest.NewServer(http.HandlerFunc(hub.HandleWS))
	defer server.Close()

	// 连接 WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket 连接失败: %v", err)
	}
	defer conn.Close()

	// 等连接注册
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Fatalf("期望 1 个客户端，实际 %d", hub.ClientCount())
	}

	// 广播 tick
	hub.BroadcastTick(model.Tick{
		Symbol:   "BTC/USDT",
		Price:    50000,
		Quantity: 10,
		Time:     time.Now(),
	})

	// 读取消息
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("读取消息失败: %v", err)
	}

	var msg model.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("解析消息失败: %v", err)
	}

	if msg.Type != "tick" {
		t.Fatalf("期望消息类型 tick，实际 %s", msg.Type)
	}

	t.Logf("WebSocket 推送验证通过: type=%s data=%s", msg.Type, string(data))
}
