package service

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourname/AccountService/model"
)

var (
	ErrInsufficientBalance = errors.New("可用余额不足")
	ErrInsufficientFrozen  = errors.New("冻结余额不足")
	ErrAccountNotFound     = errors.New("账户不存在")
)

// AccountService 账户服务（内存版，后续接 PostgreSQL）
type AccountService struct {
	mu       sync.Mutex
	accounts map[string]*model.Account // key: "userID:asset"
	ledger   []model.LedgerEntry       // 全部流水
	seqID    int64                     // 流水自增ID
}

func New() *AccountService {
	return &AccountService{
		accounts: make(map[string]*model.Account),
	}
}

// EnsureAccount 确保用户有足够资金（开发阶段自动充值，生产环境删除此方法）
func (s *AccountService) EnsureAccount(userID, asset string, minAmount int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc := s.getOrCreate(userID, asset)
	if acc.Available == 0 && acc.Frozen == 0 {
		before := acc.Available
		acc.Available = minAmount
		s.addLedger(userID, asset, model.Deposit, minAmount, before, acc.Available, "auto-init")
		log.Printf("[AccountService] 自动初始化账户: %s %s=%d", userID, asset, minAmount)
	}
}

func accountKey(userID, asset string) string {
	return userID + ":" + asset
}

// GetOrCreateAccount 获取账户，不存在则创建
func (s *AccountService) GetOrCreateAccount(userID, asset string) *model.Account {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := accountKey(userID, asset)
	if acc, ok := s.accounts[key]; ok {
		return acc
	}
	acc := &model.Account{UserID: userID, Asset: asset}
	s.accounts[key] = acc
	return acc
}

// Deposit 充值
func (s *AccountService) Deposit(userID, asset string, amount int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc := s.getOrCreate(userID, asset)
	before := acc.Available
	acc.Available += amount

	s.addLedger(userID, asset, model.Deposit, amount, before, acc.Available, "")
	return nil
}

// Withdraw 提现
func (s *AccountService) Withdraw(userID, asset string, amount int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc := s.getOrCreate(userID, asset)
	if acc.Available < amount {
		return ErrInsufficientBalance
	}
	before := acc.Available
	acc.Available -= amount

	s.addLedger(userID, asset, model.Withdraw, amount, before, acc.Available, "")
	return nil
}

// Freeze 冻结资金（下单时调用）
func (s *AccountService) Freeze(userID, asset string, amount int64, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc := s.getOrCreate(userID, asset)
	if acc.Available < amount {
		return ErrInsufficientBalance
	}
	before := acc.Available
	acc.Available -= amount
	acc.Frozen += amount

	s.addLedger(userID, asset, model.Freeze, amount, before, acc.Available, orderID)
	return nil
}

// Unfreeze 解冻资金（取消订单时调用）
func (s *AccountService) Unfreeze(userID, asset string, amount int64, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc := s.getOrCreate(userID, asset)
	if acc.Frozen < amount {
		return ErrInsufficientFrozen
	}
	before := acc.Available
	acc.Frozen -= amount
	acc.Available += amount

	s.addLedger(userID, asset, model.Unfreeze, amount, before, acc.Available, orderID)
	return nil
}

// SettleBuy 买入成交结算：扣冻结的报价资产，增加基础资产
// 例：BTC/USDT 买入成交 → 扣冻结的 USDT，增加 BTC
func (s *AccountService) SettleBuy(userID string, quoteAsset string, quoteCost int64, baseAsset string, baseAmount int64, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	quoteAcc := s.getOrCreate(userID, quoteAsset)
	if quoteAcc.Frozen < quoteCost {
		return ErrInsufficientFrozen
	}

	// 扣冻结的报价资产
	beforeQuote := quoteAcc.Available
	quoteAcc.Frozen -= quoteCost
	s.addLedger(userID, quoteAsset, model.TradeBuy, quoteCost, beforeQuote, quoteAcc.Available, orderID)

	// 增加基础资产
	baseAcc := s.getOrCreate(userID, baseAsset)
	beforeBase := baseAcc.Available
	baseAcc.Available += baseAmount
	s.addLedger(userID, baseAsset, model.TradeBuy, baseAmount, beforeBase, baseAcc.Available, orderID)

	return nil
}

// SettleSell 卖出成交结算：扣冻结的基础资产，增加报价资产
// 例：BTC/USDT 卖出成交 → 扣冻结的 BTC，增加 USDT
func (s *AccountService) SettleSell(userID string, baseAsset string, baseAmount int64, quoteAsset string, quoteGain int64, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	baseAcc := s.getOrCreate(userID, baseAsset)
	if baseAcc.Frozen < baseAmount {
		return ErrInsufficientFrozen
	}

	// 扣冻结的基础资产
	beforeBase := baseAcc.Available
	baseAcc.Frozen -= baseAmount
	s.addLedger(userID, baseAsset, model.TradeSell, baseAmount, beforeBase, baseAcc.Available, orderID)

	// 增加报价资产
	quoteAcc := s.getOrCreate(userID, quoteAsset)
	beforeQuote := quoteAcc.Available
	quoteAcc.Available += quoteGain
	s.addLedger(userID, quoteAsset, model.TradeSell, quoteGain, beforeQuote, quoteAcc.Available, orderID)

	return nil
}

// GetLedger 获取某用户某资产的全部流水
func (s *AccountService) GetLedger(userID, asset string) []model.LedgerEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	var entries []model.LedgerEntry
	for _, e := range s.ledger {
		if e.UserID == userID && e.Asset == asset {
			entries = append(entries, e)
		}
	}
	return entries
}

// --- 内部方法 ---

func (s *AccountService) getOrCreate(userID, asset string) *model.Account {
	key := accountKey(userID, asset)
	if acc, ok := s.accounts[key]; ok {
		return acc
	}
	acc := &model.Account{UserID: userID, Asset: asset}
	s.accounts[key] = acc
	return acc
}

func (s *AccountService) addLedger(userID, asset string, typ model.LedgerType, amount, before, after int64, orderID string) {
	s.seqID++
	s.ledger = append(s.ledger, model.LedgerEntry{
		ID:        fmt.Sprintf("L%d", s.seqID),
		UserID:    userID,
		Asset:     asset,
		Type:      typ,
		Amount:    amount,
		Before:    before,
		After:     after,
		OrderID:   orderID,
		CreatedAt: time.Now(),
	})
}
