package model

import "time"

// TradeMessage 成交回报（从 Kafka trades topic 消费）
type TradeMessage struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	BuyOrderID  string    `json:"buy_order_id"`
	BuyUserID   string    `json:"buy_user_id"`
	SellOrderID string    `json:"sell_order_id"`
	SellUserID  string    `json:"sell_user_id"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
}

// KLine K线（蜡烛图）
type KLine struct {
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"` // "1m", "5m", "1h"
	Open      int64     `json:"open"`
	High      int64     `json:"high"`
	Low       int64     `json:"low"`
	Close     int64     `json:"close"`
	Volume    int64     `json:"volume"`    // 成交量
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// Tick 最新成交推送
type Tick struct {
	Symbol   string    `json:"symbol"`
	Price    int64     `json:"price"`
	Quantity int64     `json:"quantity"`
	Time     time.Time `json:"time"`
}

// WSMessage WebSocket 推送消息
type WSMessage struct {
	Type string `json:"type"` // "tick", "kline"
	Data any    `json:"data"`
}
