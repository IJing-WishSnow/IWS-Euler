package engine

import (
	"container/heap"
	"time"

	"github.com/yourname/IWS-MatchingEngine/model"
)

// buyHeap 买单堆：价格高的优先，同价格时间早的优先
type buyHeap []*model.Order

func (h buyHeap) Len() int      { return len(h) }
func (h buyHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h buyHeap) Less(i, j int) bool {
	if h[i].Price != h[j].Price {
		return h[i].Price > h[j].Price // 价格高的优先
	}
	return h[i].CreatedAt.Before(h[j].CreatedAt) // 同价格，时间早的优先
}
func (h *buyHeap) Push(x any) { *h = append(*h, x.(*model.Order)) }
func (h *buyHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// sellHeap 卖单堆：价格低的优先，同价格时间早的优先
type sellHeap []*model.Order

func (h sellHeap) Len() int      { return len(h) }
func (h sellHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h sellHeap) Less(i, j int) bool {
	if h[i].Price != h[j].Price {
		return h[i].Price < h[j].Price // 价格低的优先
	}
	return h[i].CreatedAt.Before(h[j].CreatedAt) // 同价格，时间早的优先
}
func (h *sellHeap) Push(x any) { *h = append(*h, x.(*model.Order)) }
func (h *sellHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// Engine 撮合引擎，单线程串行处理
type Engine struct {
	symbol    string
	buyOrders  *buyHeap
	sellOrders *sellHeap
}

func New(symbol string) *Engine {
	bh := &buyHeap{}
	sh := &sellHeap{}
	heap.Init(bh)
	heap.Init(sh)
	return &Engine{
		symbol:     symbol,
		buyOrders:  bh,
		sellOrders: sh,
	}
}

// PlaceOrder 下单，返回本次产生的所有成交回报
func (e *Engine) PlaceOrder(o *model.Order) []model.Trade {
	o.Remaining = o.Quantity
	o.Status = model.Open

	if o.Type == model.Market {
		return e.matchMarket(o)
	}
	return e.matchLimit(o)
}

// CancelOrder 取消挂单（标记取消，下次撮合时跳过）
func (e *Engine) CancelOrder(orderID string) {
	// 遍历堆找到订单并标记取消
	for _, o := range *e.buyOrders {
		if o.ID == orderID {
			o.Status = model.Cancelled
			return
		}
	}
	for _, o := range *e.sellOrders {
		if o.ID == orderID {
			o.Status = model.Cancelled
			return
		}
	}
}

// matchLimit 限价单撮合
func (e *Engine) matchLimit(o *model.Order) []model.Trade {
	var trades []model.Trade

	if o.Side == model.Buy {
		// 买单：找卖单侧最低价，若 <= 买单价则成交
		for e.sellOrders.Len() > 0 && o.Remaining > 0 {
			best := (*e.sellOrders)[0]
			if best.Status == model.Cancelled {
				heap.Pop(e.sellOrders)
				continue
			}
			if best.Price > o.Price {
				break // 卖单最低价高于买单价，无法成交
			}
			trade := e.fill(o, best, best.Price)
			trades = append(trades, trade)
			if best.Remaining == 0 {
				heap.Pop(e.sellOrders)
			}
		}
		// 未完全成交，挂入买单堆
		if o.Remaining > 0 {
			heap.Push(e.buyOrders, o)
		}
	} else {
		// 卖单：找买单侧最高价，若 >= 卖单价则成交
		for e.buyOrders.Len() > 0 && o.Remaining > 0 {
			best := (*e.buyOrders)[0]
			if best.Status == model.Cancelled {
				heap.Pop(e.buyOrders)
				continue
			}
			if best.Price < o.Price {
				break // 买单最高价低于卖单价，无法成交
			}
			trade := e.fill(best, o, best.Price)
			trades = append(trades, trade)
			if best.Remaining == 0 {
				heap.Pop(e.buyOrders)
			}
		}
		// 未完全成交，挂入卖单堆
		if o.Remaining > 0 {
			heap.Push(e.sellOrders, o)
		}
	}

	return trades
}

// matchMarket 市价单撮合：以对手方最优价成交，不挂单
func (e *Engine) matchMarket(o *model.Order) []model.Trade {
	var trades []model.Trade

	if o.Side == model.Buy {
		for e.sellOrders.Len() > 0 && o.Remaining > 0 {
			best := (*e.sellOrders)[0]
			if best.Status == model.Cancelled {
				heap.Pop(e.sellOrders)
				continue
			}
			trade := e.fill(o, best, best.Price)
			trades = append(trades, trade)
			if best.Remaining == 0 {
				heap.Pop(e.sellOrders)
			}
		}
	} else {
		for e.buyOrders.Len() > 0 && o.Remaining > 0 {
			best := (*e.buyOrders)[0]
			if best.Status == model.Cancelled {
				heap.Pop(e.buyOrders)
				continue
			}
			trade := e.fill(best, o, best.Price)
			trades = append(trades, trade)
			if best.Remaining == 0 {
				heap.Pop(e.buyOrders)
			}
		}
	}

	// 市价单不挂单，剩余未成交部分直接取消
	if o.Remaining > 0 {
		o.Status = model.Cancelled
	}

	return trades
}

// fill 执行一次成交，更新双方订单状态，返回成交回报
func (e *Engine) fill(buyOrder, sellOrder *model.Order, price int64) model.Trade {
	qty := min(buyOrder.Remaining, sellOrder.Remaining)

	buyOrder.Remaining -= qty
	sellOrder.Remaining -= qty

	if buyOrder.Remaining == 0 {
		buyOrder.Status = model.Filled
	} else {
		buyOrder.Status = model.PartiallyFilled
	}
	if sellOrder.Remaining == 0 {
		sellOrder.Status = model.Filled
	} else {
		sellOrder.Status = model.PartiallyFilled
	}

	return model.Trade{
		ID:          buyOrder.ID + "-" + sellOrder.ID,
		Symbol:      e.symbol,
		BuyOrderID:  buyOrder.ID,
		SellOrderID: sellOrder.ID,
		Price:       price,
		Quantity:    qty,
		CreatedAt:   time.Now(),
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
