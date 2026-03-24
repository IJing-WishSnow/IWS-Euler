package model

import (
	"fmt"
	"time"
)

// Side 订单方向
type Side int

const (
	Buy Side = iota
	Sell
)

func (s Side) MarshalJSON() ([]byte, error) {
	switch s {
	case Buy:
		return []byte(`"buy"`), nil
	case Sell:
		return []byte(`"sell"`), nil
	}
	return nil, fmt.Errorf("unknown side: %d", s)
}

func (s *Side) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case `"buy"`:
		*s = Buy
	case `"sell"`:
		*s = Sell
	default:
		return fmt.Errorf("unknown side: %s", data)
	}
	return nil
}

// OrderType 订单类型
type OrderType int

const (
	Limit OrderType = iota
	Market
)

func (t OrderType) MarshalJSON() ([]byte, error) {
	switch t {
	case Limit:
		return []byte(`"limit"`), nil
	case Market:
		return []byte(`"market"`), nil
	}
	return nil, fmt.Errorf("unknown type: %d", t)
}

func (t *OrderType) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case `"limit"`:
		*t = Limit
	case `"market"`:
		*t = Market
	default:
		return fmt.Errorf("unknown type: %s", data)
	}
	return nil
}

// OrderRequest 下单请求（发到 Kafka orders topic）
type OrderRequest struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      Side      `json:"side"`
	Type      OrderType `json:"type"`
	Price     int64     `json:"price"`
	Quantity  int64     `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

// TradeResult 成交回报（发到 Kafka trades topic）
type TradeResult struct {
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
