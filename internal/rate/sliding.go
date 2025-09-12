package rate

import (
	"context"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go/valkeycompat"
)

// AllowSlidingWindow uses a sorted-set based sliding window rate limiter.
// rdb should be a valkeycompat adapter (e.g., valkeycompat.NewAdapter(client)).
func AllowSlidingWindow(rdb valkeycompat.Cmdable, key string, window time.Duration, limit int) (bool, int, int, error) {
	now := time.Now()
	cutoff := now.Add(-window)
	ctx := context.Background()

	pipe := rdb.TxPipeline() // MULTI/EXEC-style pipeline. :contentReference[oaicite:0]{index=0}
	zkey := "rl:" + key

	// 1) drop old entries
	pipe.ZRemRangeByScore(ctx, zkey, "-inf", "("+floatToStr(float64(cutoff.UnixNano()))) // :contentReference[oaicite:1]{index=1}

	// 2) add current hit
	pipe.ZAdd(ctx, zkey, valkeycompat.Z{
		Score:  float64(now.UnixNano()),
		Member: now.String(),
	}) // :contentReference[oaicite:2]{index=2}

	// 3) count
	cnt := pipe.ZCard(ctx, zkey) // :contentReference[oaicite:3]{index=3}

	// 4) keep TTL fresh
	pipe.Expire(ctx, zkey, window) // :contentReference[oaicite:4]{index=4}

	// Execute atomically
	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, int(window.Seconds()), err
	}

	c, _ := cnt.Result()
	allowed := c <= int64(limit)
	remaining := limit - int(c)
	if remaining < 0 {
		remaining = 0
	}
	return allowed, remaining, int(window.Seconds()), nil
}

func floatToStr(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
