package model

import "time"

// LedgerType 流水类型
type LedgerType int

const (
	Deposit    LedgerType = iota // 充值
	Withdraw                     // 提现
	Freeze                       // 冻结（下单时）
	Unfreeze                     // 解冻（取消订单时）
	TradeBuy                     // 买入成交（扣冻结，加资产）
	TradeSell                    // 卖出成交（扣冻结，加资金）
)

// Account 用户某个资产的账户
type Account struct {
	UserID    string
	Asset     string // 资产类型，如 "USDT"、"BTC"
	Available int64  // 可用余额
	Frozen    int64  // 冻结金额
}

// Total 总余额 = 可用 + 冻结
func (a *Account) Total() int64 {
	return a.Available + a.Frozen
}

// LedgerEntry 资金流水（每一笔变动都记录，可审计可回溯）
type LedgerEntry struct {
	ID        string
	UserID    string
	Asset     string
	Type      LedgerType
	Amount    int64     // 变动金额（正数）
	Before    int64     // 变动前可用余额
	After     int64     // 变动后可用余额
	OrderID   string    // 关联订单ID（如有）
	CreatedAt time.Time
}
