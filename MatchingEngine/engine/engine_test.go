package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/yourname/MatchingEngine/model"
)

func newOrder(id string, side model.Side, orderType model.OrderType, price, qty int64) *model.Order {
	return &model.Order{
		ID:        id,
		UserID:    "user-" + id,
		Symbol:    "BTC/USDT",
		Side:      side,
		Type:      orderType,
		Price:     price,
		Quantity:  qty,
		CreatedAt: time.Now(),
	}
}

// 测试1：限价单完全成交
func TestLimitOrderFullFill(t *testing.T) {
	e := New("BTC/USDT")

	// 先挂一个卖单：价格 100，数量 10
	e.PlaceOrder(newOrder("sell-1", model.Sell, model.Limit, 100, 10))

	// 再下一个买单：价格 100，数量 10，应该完全成交
	trades := e.PlaceOrder(newOrder("buy-1", model.Buy, model.Limit, 100, 10))

	if len(trades) != 1 {
		t.Fatalf("期望 1 笔成交，实际 %d 笔", len(trades))
	}
	if trades[0].Quantity != 10 {
		t.Fatalf("期望成交量 10，实际 %d", trades[0].Quantity)
	}
	if trades[0].Price != 100 {
		t.Fatalf("期望成交价 100，实际 %d", trades[0].Price)
	}
}

// 测试2：限价单部分成交
func TestLimitOrderPartialFill(t *testing.T) {
	e := New("BTC/USDT")

	// 卖单：数量 5
	e.PlaceOrder(newOrder("sell-1", model.Sell, model.Limit, 100, 5))

	// 买单：数量 10，只能成交 5，剩余 5 挂单
	trades := e.PlaceOrder(newOrder("buy-1", model.Buy, model.Limit, 100, 10))

	if len(trades) != 1 {
		t.Fatalf("期望 1 笔成交，实际 %d 笔", len(trades))
	}
	if trades[0].Quantity != 5 {
		t.Fatalf("期望成交量 5，实际 %d", trades[0].Quantity)
	}
	// 买单剩余 5 应该还在订单簿里
	if e.buyOrders.Len() != 1 {
		t.Fatalf("期望买单堆有 1 个挂单，实际 %d", e.buyOrders.Len())
	}
}

// 测试3：价格不匹配，无法成交
func TestLimitOrderNoMatch(t *testing.T) {
	e := New("BTC/USDT")

	// 卖单价格 200，买单价格 100，无法成交
	e.PlaceOrder(newOrder("sell-1", model.Sell, model.Limit, 200, 10))
	trades := e.PlaceOrder(newOrder("buy-1", model.Buy, model.Limit, 100, 10))

	if len(trades) != 0 {
		t.Fatalf("期望 0 笔成交，实际 %d 笔", len(trades))
	}
	// 双方都应该挂单
	if e.buyOrders.Len() != 1 || e.sellOrders.Len() != 1 {
		t.Fatal("买卖双方都应该挂单")
	}
}

// 测试4：市价单成交
func TestMarketOrder(t *testing.T) {
	e := New("BTC/USDT")

	// 挂两个卖单
	e.PlaceOrder(newOrder("sell-1", model.Sell, model.Limit, 100, 5))
	e.PlaceOrder(newOrder("sell-2", model.Sell, model.Limit, 101, 5))

	// 市价买单，数量 8，应该先吃掉 sell-1 的 5，再吃 sell-2 的 3
	trades := e.PlaceOrder(newOrder("buy-1", model.Buy, model.Market, 0, 8))

	if len(trades) != 2 {
		t.Fatalf("期望 2 笔成交，实际 %d 笔", len(trades))
	}
	if trades[0].Quantity != 5 || trades[1].Quantity != 3 {
		t.Fatalf("期望成交量 5 和 3，实际 %d 和 %d", trades[0].Quantity, trades[1].Quantity)
	}
}

// 测试5：价格优先级，高价买单先成交
func TestBuyPricePriority(t *testing.T) {
	e := New("BTC/USDT")

	// 两个买单，价格不同
	e.PlaceOrder(newOrder("buy-1", model.Buy, model.Limit, 100, 10))
	e.PlaceOrder(newOrder("buy-2", model.Buy, model.Limit, 200, 10))

	// 卖单，应该先和价格高的 buy-2 成交
	trades := e.PlaceOrder(newOrder("sell-1", model.Sell, model.Limit, 100, 10))

	if len(trades) != 1 {
		t.Fatalf("期望 1 笔成交，实际 %d 笔", len(trades))
	}
	if trades[0].BuyOrderID != "buy-2" {
		t.Fatalf("期望和 buy-2 成交，实际和 %s 成交", trades[0].BuyOrderID)
	}
}

// 测试6：资产守恒（fuzz 思路的简化版）
// 大量随机订单后，买方总成交量 == 卖方总成交量
func TestAssetConservation(t *testing.T) {
	e := New("BTC/USDT")

	orders := []*model.Order{
		newOrder("b1", model.Buy, model.Limit, 105, 3),
		newOrder("s1", model.Sell, model.Limit, 100, 2),
		newOrder("b2", model.Buy, model.Limit, 102, 5),
		newOrder("s2", model.Sell, model.Limit, 101, 4),
		newOrder("s3", model.Sell, model.Limit, 103, 6),
		newOrder("b3", model.Buy, model.Limit, 104, 2),
	}

	var totalBuyQty, totalSellQty int64
	for _, o := range orders {
		trades := e.PlaceOrder(o)
		for _, tr := range trades {
			totalBuyQty += tr.Quantity
			totalSellQty += tr.Quantity
		}
	}

	if totalBuyQty != totalSellQty {
		t.Fatalf("资产不守恒：买方成交 %d，卖方成交 %d", totalBuyQty, totalSellQty)
	}
	fmt.Printf("资产守恒验证通过，总成交量：%d\n", totalBuyQty)
}

// Benchmark：测试撮合引擎每秒能处理多少笔订单
func BenchmarkPlaceOrder(b *testing.B) {
	e := New("BTC/USDT")
	// 预先挂一批卖单，保证买单能持续成交
	for i := 0; i < 1000; i++ {
		e.PlaceOrder(newOrder(
			fmt.Sprintf("sell-%d", i),
			model.Sell, model.Limit,
			int64(100+i), 100,
		))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.PlaceOrder(newOrder(
			fmt.Sprintf("buy-%d", i),
			model.Buy, model.Limit,
			int64(100+i%1000), 1,
		))
	}
}
