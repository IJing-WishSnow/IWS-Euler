package kline

import (
	"testing"
	"time"

	"github.com/yourname/IWS-MarketData/model"
)

func TestAggregatorFeed(t *testing.T) {
	var completed []model.KLine
	agg := New(func(k model.KLine) {
		completed = append(completed, k)
	})

	// 在同一分钟内喂入多笔成交
	base := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	agg.Feed("BTC/USDT", 50000, 10, base)
	agg.Feed("BTC/USDT", 51000, 5, base.Add(10*time.Second))
	agg.Feed("BTC/USDT", 49000, 8, base.Add(20*time.Second))
	agg.Feed("BTC/USDT", 50500, 3, base.Add(30*time.Second))

	// 检查当前 1m K 线
	k := agg.GetCurrent("BTC/USDT", "1m")
	if k == nil {
		t.Fatal("1m K 线不存在")
	}
	if k.Open != 50000 {
		t.Fatalf("Open 期望 50000，实际 %d", k.Open)
	}
	if k.High != 51000 {
		t.Fatalf("High 期望 51000，实际 %d", k.High)
	}
	if k.Low != 49000 {
		t.Fatalf("Low 期望 49000，实际 %d", k.Low)
	}
	if k.Close != 50500 {
		t.Fatalf("Close 期望 50500，实际 %d", k.Close)
	}
	if k.Volume != 26 {
		t.Fatalf("Volume 期望 26，实际 %d", k.Volume)
	}

	t.Logf("1m K线: O=%d H=%d L=%d C=%d V=%d", k.Open, k.High, k.Low, k.Close, k.Volume)

	// 喂入下一分钟的数据，触发上一根 K 线归档
	agg.Feed("BTC/USDT", 52000, 7, base.Add(61*time.Second))

	if len(completed) == 0 {
		t.Fatal("K 线归档回调未触发")
	}

	archived := completed[0]
	if archived.Close != 50500 {
		t.Fatalf("归档 K 线 Close 期望 50500，实际 %d", archived.Close)
	}

	t.Logf("归档 K线: O=%d H=%d L=%d C=%d V=%d", archived.Open, archived.High, archived.Low, archived.Close, archived.Volume)
}
