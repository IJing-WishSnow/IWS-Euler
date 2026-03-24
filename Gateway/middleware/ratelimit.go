package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter Redis 滑动窗口限流
type RateLimiter struct {
	client *redis.Client
	limit  int           // 窗口内最大请求数
	window time.Duration // 窗口大小
}

func NewRateLimiter(addr string, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: redis.NewClient(&redis.Options{Addr: addr}),
		limit:  limit,
		window: window,
	}
}

// Limit 限流中间件：每个用户在窗口内最多 N 次请求
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			// 未登录用户用 IP 限流
			userID = c.ClientIP()
		}

		key := "rate:" + userID.(string)
		ctx := context.Background()
		now := time.Now().UnixMilli()

		pipe := rl.client.Pipeline()
		// 移除窗口外的旧记录
		pipe.ZRemRangeByScore(ctx, key, "0", formatInt(now-rl.window.Milliseconds()))
		// 统计窗口内请求数
		countCmd := pipe.ZCard(ctx, key)
		// 添加当前请求
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
		// 设置过期时间
		pipe.Expire(ctx, key, rl.window)
		pipe.Exec(ctx)

		count := countCmd.Val()
		if count >= int64(rl.limit) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func formatInt(n int64) string {
	return strconv.FormatInt(n, 10)
}
