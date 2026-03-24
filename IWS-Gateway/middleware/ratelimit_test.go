package middleware

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestFormatInt 验证 formatInt 输出正确的十进制数字字符串（修复前使用了 time.Format 导致限流完全失效）
func TestFormatInt(t *testing.T) {
	now := time.Now().UnixMilli()
	result := formatInt(now)

	// 必须是纯数字字符串，可以被 ParseInt 解析
	parsed, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		t.Fatalf("formatInt 输出了非数字字符串: %q, err=%v", result, err)
	}
	if parsed != now {
		t.Fatalf("formatInt 输出值不匹配: expected %d, got %d (string %q)", now, parsed, result)
	}

	// 确保不是旧 bug 的日期格式（time.Format 输出如 "20240322150405123" ≈ 20 千万亿）
	// Unix ms 时间戳约 1.7 万亿，合理范围 < 3 万亿
	const maxExpected = int64(3_000_000_000_000)
	if parsed > maxExpected {
		t.Fatalf("formatInt 输出值异常大 (%d)，疑似使用日期格式而非 Unix 毫秒", parsed)
	}
	t.Logf("formatInt(%d) = %q ✓", now, result)
}

// TestFormatIntCutoff 验证滑动窗口 cutoff 计算正确
func TestFormatIntCutoff(t *testing.T) {
	window := time.Second
	now := time.Now().UnixMilli()
	cutoff := now - window.Milliseconds()

	cutoffStr := formatInt(cutoff)
	cutoffParsed, err := strconv.ParseInt(cutoffStr, 10, 64)
	if err != nil {
		t.Fatalf("cutoff formatInt 输出非数字: %q", cutoffStr)
	}
	// cutoff 必须精确
	if cutoffParsed != cutoff {
		t.Fatalf("cutoff 值不匹配: expected %d, got %d", cutoff, cutoffParsed)
	}
	// cutoff 必须小于 now（时间窗口合理性）
	if cutoffParsed >= now {
		t.Fatalf("cutoff (%d) 不应 >= now (%d)", cutoffParsed, now)
	}
	t.Logf("窗口 cutoff: %d < now: %d (diff=%dms) ✓", cutoffParsed, now, now-cutoffParsed)
}

// TestRateLimiterIntegration 真实 Redis 集成测试
// 前提：kubectl port-forward svc/redis 6379:6379 -n iws-Euler
func TestRateLimiterIntegration(t *testing.T) {
	rl := NewRateLimiter("localhost:6379", 3, time.Second)
	ctx := context.Background()

	if err := rl.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis 不可达，跳过集成测试: %v", err)
	}

	testKey := "rate:test-ratelimit-integration"
	rl.client.Del(ctx, testKey)
	defer rl.client.Del(ctx, testKey)

	limit := int64(3)
	window := time.Second

	// 模拟 3 次请求，每次都不应被限流
	for i := 0; i < 3; i++ {
		now := time.Now().UnixMilli() + int64(i) // +i 确保 member 唯一
		pipe := rl.client.Pipeline()
		pipe.ZRemRangeByScore(ctx, testKey, "0", formatInt(now-window.Milliseconds()))
		countCmd := pipe.ZCard(ctx, testKey)
		pipe.ZAdd(ctx, testKey, redis.Z{Score: float64(now), Member: now})
		pipe.Expire(ctx, testKey, window)
		pipe.Exec(ctx)

		count := countCmd.Val()
		if count >= limit {
			t.Errorf("请求 #%d 不应被限流 (count=%d >= limit=%d)", i+1, count, limit)
		}
		t.Logf("请求 #%d: 窗口内记录数 = %d ✓", i+1, count)
	}

	// 第 4 次：此时窗口内已有 3 条记录，应触发限流
	finalCount := rl.client.ZCard(ctx, testKey).Val()
	if finalCount < limit {
		t.Errorf("期望窗口内 >= %d 条记录以验证限流，实际 %d 条", limit, finalCount)
	}
	t.Logf("第 4 次请求前记录数 = %d >= limit=%d，限流逻辑有效 ✓", finalCount, limit)
}
