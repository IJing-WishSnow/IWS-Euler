package kline

import (
	"sync"
	"time"

	"github.com/yourname/MarketData/model"
)

// 支持的 K 线周期
var Intervals = map[string]time.Duration{
	"1m": time.Minute,
	"5m": 5 * time.Minute,
	"1h": time.Hour,
}

// Aggregator K线聚合器：接收成交 tick，聚合成各周期 K 线
type Aggregator struct {
	mu      sync.RWMutex
	candles map[string]*model.KLine // key: "symbol:interval"
	history map[string][]model.KLine // 历史 K 线
	onKLine func(model.KLine)        // K 线完成回调
}

func New(onKLine func(model.KLine)) *Aggregator {
	return &Aggregator{
		candles: make(map[string]*model.KLine),
		history: make(map[string][]model.KLine),
		onKLine: onKLine,
	}
}

// Feed 喂入一笔成交
func (a *Aggregator) Feed(symbol string, price, quantity int64, t time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for interval, duration := range Intervals {
		key := symbol + ":" + interval
		startTime := t.Truncate(duration) // 对齐到周期起点
		endTime := startTime.Add(duration)

		candle, exists := a.candles[key]

		// 当前 K 线已过期，归档并创建新的
		if exists && !t.Before(candle.EndTime) {
			a.archive(key, *candle)
			exists = false
		}

		if !exists {
			// 新 K 线
			a.candles[key] = &model.KLine{
				Symbol:    symbol,
				Interval:  interval,
				Open:      price,
				High:      price,
				Low:       price,
				Close:     price,
				Volume:    quantity,
				StartTime: startTime,
				EndTime:   endTime,
			}
		} else {
			// 更新当前 K 线
			if price > candle.High {
				candle.High = price
			}
			if price < candle.Low {
				candle.Low = price
			}
			candle.Close = price
			candle.Volume += quantity
		}
	}
}

// GetCurrent 获取当前未完成的 K 线
func (a *Aggregator) GetCurrent(symbol, interval string) *model.KLine {
	a.mu.RLock()
	defer a.mu.RUnlock()

	key := symbol + ":" + interval
	if c, ok := a.candles[key]; ok {
		cp := *c
		return &cp
	}
	return nil
}

// GetHistory 获取历史 K 线
func (a *Aggregator) GetHistory(symbol, interval string, limit int) []model.KLine {
	a.mu.RLock()
	defer a.mu.RUnlock()

	key := symbol + ":" + interval
	h := a.history[key]
	if len(h) <= limit {
		return h
	}
	return h[len(h)-limit:]
}

func (a *Aggregator) archive(key string, candle model.KLine) {
	a.history[key] = append(a.history[key], candle)
	// 每个周期最多保留 1000 根
	if len(a.history[key]) > 1000 {
		a.history[key] = a.history[key][1:]
	}
	if a.onKLine != nil {
		a.onKLine(candle)
	}
}
