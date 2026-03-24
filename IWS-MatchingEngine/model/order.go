package model

import "time"

// Side 订单方向
type Side int

const (
	Buy  Side = iota // 买单
	Sell             // 卖单
)

// OrderType 订单类型
type OrderType int

const (
	Limit  OrderType = iota // 限价单：指定价格
	Market                  // 市价单：以当前最优价成交
)

// OrderStatus 订单状态
type OrderStatus int

const (
	Open            OrderStatus = iota // 挂单中
	PartiallyFilled                    // 部分成交
	Filled                             // 完全成交
	Cancelled                          // 已取消
)

// Order 订单
type Order struct {
	ID        string
	UserID    string
	Symbol    string // 交易对，如 "BTC/USDT"
	Side      Side
	Type      OrderType
	Price     int64 // 价格，整数表示，避免浮点误差（如 1 USDT = 1_000_000）
	Quantity  int64 // 下单总数量
	Remaining int64 // 剩余未成交数量
	Status    OrderStatus
	CreatedAt time.Time
}

// Trade 成交回报
type Trade struct {
	ID          string
	Symbol      string
	BuyOrderID  string
	SellOrderID string
	Price       int64 // 成交价
	Quantity    int64 // 成交量
	CreatedAt   time.Time
}
