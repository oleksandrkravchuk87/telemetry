package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	tests := []struct {
		name           string
		rate           int
		initialBucket  int
		requestBytes   int
		timeSinceLast  time.Duration
		expectedResult bool
		expectedBucket int
	}{
		{
			name:           "allow when bucket has enough tokens",
			rate:           100,
			initialBucket:  100,
			requestBytes:   50,
			timeSinceLast:  0,
			expectedResult: true,
			expectedBucket: 50,
		},
		{
			name:           "deny when bucket has insufficient tokens",
			rate:           100,
			initialBucket:  30,
			requestBytes:   50,
			timeSinceLast:  0,
			expectedResult: false,
			expectedBucket: 30,
		},
		{
			name:           "allow exact token match",
			rate:           100,
			initialBucket:  50,
			requestBytes:   50,
			timeSinceLast:  0,
			expectedResult: true,
			expectedBucket: 0,
		},
		{
			name:           "refill tokens after time passes",
			rate:           100,
			initialBucket:  0,
			requestBytes:   50,
			timeSinceLast:  time.Second,
			expectedResult: true,
			expectedBucket: 50,
		},
		{
			name:           "partial refill still insufficient",
			rate:           100,
			initialBucket:  0,
			requestBytes:   80,
			timeSinceLast:  500 * time.Millisecond,
			expectedResult: false,
			expectedBucket: 50,
		},
		{
			name:           "cap bucket at rate limit",
			rate:           100,
			initialBucket:  50,
			requestBytes:   25,
			timeSinceLast:  2 * time.Second,
			expectedResult: true,
			expectedBucket: 75,
		},
		{
			name:           "zero bytes request always allowed",
			rate:           100,
			initialBucket:  0,
			requestBytes:   0,
			timeSinceLast:  0,
			expectedResult: true,
			expectedBucket: 0,
		},
		{
			name:           "large request exceeding rate",
			rate:           100,
			initialBucket:  100,
			requestBytes:   150,
			timeSinceLast:  0,
			expectedResult: false,
			expectedBucket: 100,
		},
		{
			name:           "very small time increment",
			rate:           1000,
			initialBucket:  0,
			requestBytes:   1,
			timeSinceLast:  time.Millisecond,
			expectedResult: true,
			expectedBucket: 0,
		},
		{
			name:           "very small time increment insufficient",
			rate:           1000,
			initialBucket:  0,
			requestBytes:   2,
			timeSinceLast:  time.Millisecond,
			expectedResult: false,
			expectedBucket: 1,
		},
		{
			name:           "low rate limiter",
			rate:           1,
			initialBucket:  1,
			requestBytes:   1,
			timeSinceLast:  0,
			expectedResult: true,
			expectedBucket: 0,
		},
		{
			name:           "high rate with long duration",
			rate:           10000,
			initialBucket:  0,
			requestBytes:   5000,
			timeSinceLast:  time.Minute,
			expectedResult: true,
			expectedBucket: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RateLimiter{
				rate:       tt.rate,
				bucket:     tt.initialBucket,
				lastUpdate: time.Now().Add(-tt.timeSinceLast),
			}

			result := rl.Allow(tt.requestBytes)

			if result != tt.expectedResult {
				t.Errorf("Allow() = %v, want %v", result, tt.expectedResult)
			}

			if rl.bucket != tt.expectedBucket {
				t.Errorf("bucket after Allow() = %v, want %v", rl.bucket, tt.expectedBucket)
			}
		})
	}
}

func TestRateLimiter_Allow_Concurrent(t *testing.T) {
	rl := NewRateLimiter(1000)

	var wg sync.WaitGroup
	var allowed, denied int32

	// Simulate 100 concurrent requests of 10 bytes each
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow(10) {
				atomic.AddInt32(&allowed, 1)
			} else {
				atomic.AddInt32(&denied, 1)
			}
		}()
	}

	wg.Wait()

	total := atomic.LoadInt32(&allowed) + atomic.LoadInt32(&denied)
	if total != 100 {
		t.Errorf("Expected 100 total requests, got %d", total)
	}

	// Should allow exactly 100 requests (1000 tokens / 10 bytes each)
	if atomic.LoadInt32(&allowed) != 100 {
		t.Errorf("Expected all 100 requests to be allowed initially, got %d allowed", atomic.LoadInt32(&allowed))
	}
}

func TestRateLimiter_Allow_Sequential(t *testing.T) {
	rl := NewRateLimiter(100)

	// First request should be allowed
	if !rl.Allow(50) {
		t.Error("First request should be allowed")
	}

	// Second request should be allowed
	if !rl.Allow(50) {
		t.Error("Second request should be allowed")
	}

	// Third request should be denied (bucket empty)
	if rl.Allow(1) {
		t.Error("Third request should be denied")
	}

	// Wait for refill and try again
	time.Sleep(time.Second)
	if !rl.Allow(50) {
		t.Error("Request after refill should be allowed")
	}
}

func TestRateLimiter_Allow_EdgeCases(t *testing.T) {
	t.Run("negative bytes", func(t *testing.T) {
		rl := NewRateLimiter(100)
		// Negative bytes should always be allowed and increase bucket
		if !rl.Allow(-10) {
			t.Error("Negative bytes request should be allowed")
		}
		if rl.bucket != 110 {
			t.Errorf("Expected bucket to be 110, got %d", rl.bucket)
		}
	})

	t.Run("zero rate limiter", func(t *testing.T) {
		rl := NewRateLimiter(0)
		// With zero rate, only zero-byte requests should be allowed
		if !rl.Allow(0) {
			t.Error("Zero bytes should be allowed even with zero rate")
		}
		if rl.Allow(1) {
			t.Error("Non-zero bytes should be denied with zero rate")
		}
	})

	t.Run("very large time gap", func(t *testing.T) {
		rl := &RateLimiter{
			rate:       100,
			bucket:     0,
			lastUpdate: time.Now().Add(-24 * time.Hour),
		}

		if !rl.Allow(100) {
			t.Error("Request after very long time should be allowed")
		}

		// Bucket should be capped at rate
		if rl.bucket != 0 {
			t.Errorf("Expected bucket to be 0 after consuming all tokens, got %d", rl.bucket)
		}
	})
}
