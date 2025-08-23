package ratelimit

import (
	"sync"
	"time"
)

type RateLimiter struct {
	rate       int
	bucket     int
	lastUpdate time.Time
	mu         sync.Mutex
}

func NewRateLimiter(rate int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		bucket:     rate,
		lastUpdate: time.Now(),
	}
}

func (rl *RateLimiter) Allow(bytes int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	sinceLast := now.Sub(rl.lastUpdate)

	tokensToAdd := int(sinceLast.Seconds() * float64(rl.rate))
	rl.bucket += tokensToAdd

	if rl.bucket > rl.rate {
		rl.bucket = rl.rate
	}

	rl.lastUpdate = now

	if rl.bucket >= bytes {
		rl.bucket -= bytes
		return true
	}

	return false
}
