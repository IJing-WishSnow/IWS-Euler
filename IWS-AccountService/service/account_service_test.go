package service

import (
	"testing"
)

// 测试1：充值和提现
func TestDepositAndWithdraw(t *testing.T) {
	s := New()

	// 充值 1000 USDT
	s.Deposit("user1", "USDT", 1000)
	acc := s.GetOrCreateAccount("user1", "USDT")
	if acc.Available != 1000 {
		t.Fatalf("期望可用 1000，实际 %d", acc.Available)
	}

	// 提现 300
	s.Withdraw("user1", "USDT", 300)
	if acc.Available != 700 {
		t.Fatalf("期望可用 700，实际 %d", acc.Available)
	}

	// 提现超额，应该失败
	err := s.Withdraw("user1", "USDT", 800)
	if err != ErrInsufficientBalance {
		t.Fatal("超额提现应该返回余额不足错误")
	}
}

// 测试2：冻结和解冻
func TestFreezeAndUnfreeze(t *testing.T) {
	s := New()
	s.Deposit("user1", "USDT", 1000)

	// 冻结 400（模拟下单）
	s.Freeze("user1", "USDT", 400, "order-1")
	acc := s.GetOrCreateAccount("user1", "USDT")
	if acc.Available != 600 || acc.Frozen != 400 {
		t.Fatalf("期望可用 600 冻结 400，实际可用 %d 冻结 %d", acc.Available, acc.Frozen)
	}
	// 总余额不变
	if acc.Total() != 1000 {
		t.Fatalf("总余额应该不变，期望 1000，实际 %d", acc.Total())
	}

	// 解冻 400（模拟取消订单）
	s.Unfreeze("user1", "USDT", 400, "order-1")
	if acc.Available != 1000 || acc.Frozen != 0 {
		t.Fatalf("解冻后期望可用 1000 冻结 0，实际可用 %d 冻结 %d", acc.Available, acc.Frozen)
	}
}

// 测试3：买入成交结算
// 场景：用户用 USDT 买 BTC，价格 100，数量 5
func TestSettleBuy(t *testing.T) {
	s := New()
	s.Deposit("user1", "USDT", 1000)

	// 下单冻结 500 USDT（价格 100 × 数量 5）
	s.Freeze("user1", "USDT", 500, "order-1")

	// 成交结算：扣 500 冻结 USDT，加 5 BTC
	err := s.SettleBuy("user1", "USDT", 500, "BTC", 5, "order-1")
	if err != nil {
		t.Fatalf("结算失败：%v", err)
	}

	usdt := s.GetOrCreateAccount("user1", "USDT")
	btc := s.GetOrCreateAccount("user1", "BTC")

	if usdt.Available != 500 || usdt.Frozen != 0 {
		t.Fatalf("USDT 期望可用 500 冻结 0，实际可用 %d 冻结 %d", usdt.Available, usdt.Frozen)
	}
	if btc.Available != 5 {
		t.Fatalf("BTC 期望可用 5，实际 %d", btc.Available)
	}
}

// 测试4：卖出成交结算
// 场景：用户卖 BTC 换 USDT，价格 100，数量 3
func TestSettleSell(t *testing.T) {
	s := New()
	s.Deposit("user1", "BTC", 10)

	// 下单冻结 3 BTC
	s.Freeze("user1", "BTC", 3, "order-2")

	// 成交结算：扣 3 冻结 BTC，加 300 USDT
	err := s.SettleSell("user1", "BTC", 3, "USDT", 300, "order-2")
	if err != nil {
		t.Fatalf("结算失败：%v", err)
	}

	btc := s.GetOrCreateAccount("user1", "BTC")
	usdt := s.GetOrCreateAccount("user1", "USDT")

	if btc.Available != 7 || btc.Frozen != 0 {
		t.Fatalf("BTC 期望可用 7 冻结 0，实际可用 %d 冻结 %d", btc.Available, btc.Frozen)
	}
	if usdt.Available != 300 {
		t.Fatalf("USDT 期望可用 300，实际 %d", usdt.Available)
	}
}

// 测试5：流水审计 — 每笔操作都有记录
func TestLedgerAudit(t *testing.T) {
	s := New()
	s.Deposit("user1", "USDT", 1000)
	s.Freeze("user1", "USDT", 200, "order-1")
	s.Unfreeze("user1", "USDT", 200, "order-1")

	entries := s.GetLedger("user1", "USDT")
	if len(entries) != 3 {
		t.Fatalf("期望 3 条流水，实际 %d 条", len(entries))
	}
}

// 测试6：资产守恒 — 完整交易流程后总资产不变
func TestAssetConservation(t *testing.T) {
	s := New()

	// 两个用户，初始资产
	s.Deposit("buyer", "USDT", 10000)
	s.Deposit("seller", "BTC", 100)

	// 买方下单冻结 500 USDT
	s.Freeze("buyer", "USDT", 500, "order-1")
	// 卖方下单冻结 5 BTC
	s.Freeze("seller", "BTC", 5, "order-2")

	// 成交：价格 100，数量 5
	s.SettleBuy("buyer", "USDT", 500, "BTC", 5, "order-1")
	s.SettleSell("seller", "BTC", 5, "USDT", 500, "order-2")

	// 验证：系统总 USDT = 10000，总 BTC = 100
	buyerUSDT := s.GetOrCreateAccount("buyer", "USDT")
	sellerUSDT := s.GetOrCreateAccount("seller", "USDT")
	buyerBTC := s.GetOrCreateAccount("buyer", "BTC")
	sellerBTC := s.GetOrCreateAccount("seller", "BTC")

	totalUSDT := buyerUSDT.Total() + sellerUSDT.Total()
	totalBTC := buyerBTC.Total() + sellerBTC.Total()

	if totalUSDT != 10000 {
		t.Fatalf("USDT 不守恒：期望 10000，实际 %d", totalUSDT)
	}
	if totalBTC != 100 {
		t.Fatalf("BTC 不守恒：期望 100，实际 %d", totalBTC)
	}
}
