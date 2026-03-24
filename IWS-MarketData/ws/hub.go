package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yourname/IWS-MarketData/model"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // 开发阶段允许所有来源
}

// Hub WebSocket 连接管理器
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
	}
}

// HandleWS 处理 WebSocket 连接请求
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] 升级连接失败: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	log.Printf("[WS] 新连接，当前 %d 个客户端", len(h.clients))

	// 读循环（检测断开）
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
			log.Printf("[WS] 连接断开，剩余 %d 个客户端", len(h.clients))
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

// BroadcastTick 广播最新成交
func (h *Hub) BroadcastTick(tick model.Tick) {
	msg := model.WSMessage{Type: "tick", Data: tick}
	h.broadcast(msg)
}

// BroadcastKLine 广播 K 线更新
func (h *Hub) BroadcastKLine(kline model.KLine) {
	msg := model.WSMessage{Type: "kline", Data: kline}
	h.broadcast(msg)
}

func (h *Hub) broadcast(msg model.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	var dead []*websocket.Conn

	h.mu.RLock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			dead = append(dead, conn)
		}
	}
	h.mu.RUnlock()

	if len(dead) > 0 {
		h.mu.Lock()
		for _, conn := range dead {
			conn.Close()
			delete(h.clients, conn)
		}
		h.mu.Unlock()
	}
}

// ClientCount 当前连接数
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
