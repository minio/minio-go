package minio

import (
	"context"
	"math/rand"
	"testing"
	"time"
)

func TestRetryTimer(t *testing.T) {
	t.Run("withLimit", func(t *testing.T) {
		t.Parallel()
		c := &Client{random: rand.New(rand.NewSource(42))}
		ctx := context.Background()
		var count int
		for range c.newRetryTimer(ctx, 3, time.Millisecond, 10*time.Millisecond, 0.0) {
			count++
		}
		if count != 3 {
			t.Errorf("expected exactly 3 yields")
		}
	})

	t.Run("checkDelay", func(t *testing.T) {
		t.Parallel()
		c := &Client{random: rand.New(rand.NewSource(42))}
		ctx := context.Background()
		prev := time.Now()
		baseSleep := time.Millisecond
		maxSleep := 10 * time.Millisecond
		for i := range c.newRetryTimer(ctx, 6, baseSleep, maxSleep, 0.0) {
			if i == 0 {
				// there is no sleep for the first execution
				if time.Since(prev) >= time.Millisecond {
					t.Errorf("expected to not sleep for the first instance of the loop")
				}
				prev = time.Now()
				continue
			}
			expect := baseSleep * time.Duration(1<<uint(i-1))
			expect = min(expect, maxSleep)
			if d := time.Since(prev); d < expect || d > 2*maxSleep {
				t.Errorf("expected to sleep for at least %s", expect.String())
			}
			prev = time.Now()
		}
	})

	t.Run("withBreak", func(t *testing.T) {
		t.Parallel()
		c := &Client{random: rand.New(rand.NewSource(42))}
		ctx := context.Background()
		var count int
		for range c.newRetryTimer(ctx, 10, time.Millisecond, 10*time.Millisecond, 0.5) {
			count++
			if count >= 3 {
				break
			}
		}
		if count != 3 {
			t.Errorf("expected exactly 3 yields")
		}
	})

	t.Run("withCancelledContext", func(t *testing.T) {
		t.Parallel()
		c := &Client{random: rand.New(rand.NewSource(42))}
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		var count int
		for range c.newRetryTimer(ctx, 10, time.Millisecond, 10*time.Millisecond, 0.5) {
			count++
		}
		if count != 0 {
			t.Errorf("expected no yields")
		}
	})
	t.Run("whileCancelledContext", func(t *testing.T) {
		t.Parallel()
		c := &Client{random: rand.New(rand.NewSource(42))}
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		var count int
		for range c.newRetryTimer(ctx, 10, time.Millisecond, 10*time.Millisecond, 0.5) {
			count++
			cancel()
		}
		cancel()
		if count != 1 {
			t.Errorf("expected only one yield")
		}
	})
}

func TestRetryContinuous(t *testing.T) {
	t.Run("checkDelay", func(t *testing.T) {
		t.Parallel()
		c := &Client{random: rand.New(rand.NewSource(42))}
		prev := time.Now()
		baseSleep := time.Millisecond
		maxSleep := 10 * time.Millisecond
		for i := range c.newRetryTimerContinous(time.Millisecond, 10*time.Millisecond, 0.0) {
			if i == 0 {
				// there is no sleep for the first execution
				if time.Since(prev) >= time.Millisecond {
					t.Errorf("expected to not sleep for the first instance of the loop")
				}
				prev = time.Now()
				continue
			}
			expect := baseSleep * time.Duration(1<<uint(i-1))
			expect = min(expect, maxSleep)
			if d := time.Since(prev); d < expect || d > 2*maxSleep {
				t.Errorf("expected to sleep for at least %s", expect.String())
			}
			prev = time.Now()
			if i >= 10 {
				break
			}
		}
	})

	t.Run("withBreak", func(t *testing.T) {
		t.Parallel()
		c := &Client{random: rand.New(rand.NewSource(42))}
		var count int
		for range c.newRetryTimerContinous(time.Millisecond, 10*time.Millisecond, 0.5) {
			count++
			if count >= 3 {
				break
			}
		}
		if count != 3 {
			t.Errorf("expected exactly 3 yields")
		}
	})
}
