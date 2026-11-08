package concurrency

import (
	"context"
	"net/netip"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestKeyedLimiterEnforcesBurstPerKey(t *testing.T) {
	t.Parallel()

	l := NewKeyedLimiter[netip.Addr](rate.Every(time.Hour), 3, 1000, time.Minute)
	a := netip.MustParseAddr("1.2.3.4")
	b := netip.MustParseAddr("5.6.7.8")

	for range 3 {
		if !l.Allow(a) {
			t.Fatal("expected allow within burst")
		}
	}

	if l.Allow(a) {
		t.Fatal("expected deny after burst exhausted")
	}

	if !l.Allow(b) {
		t.Fatal("other key must be independent")
	}
}

func TestKeyedLimiterStringKeysAndWait(t *testing.T) {
	t.Parallel()

	l := NewKeyedLimiter[string](rate.Every(time.Hour), 1, 10, time.Minute)
	if !l.Allow("x") || l.Allow("x") || !l.Allow("y") {
		t.Fatal("unexpected allow results")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if err := l.Wait(ctx, "x"); err == nil {
		t.Fatal("expected wait to fail when rate exhausted")
	}

	if err := l.Wait(context.Background(), "z"); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

func TestKeyedLimiterConcurrent(t *testing.T) {
	t.Parallel()

	l := NewKeyedLimiter[netip.Addr](rate.Every(time.Hour), 5, 100_000, time.Minute)

	var wg sync.WaitGroup

	var mu sync.Mutex

	total := 0

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			count := 0

			for n := range 200 {
				addr := netip.AddrFrom4([4]byte{10, 0, byte(n / 256), byte(n)})
				if l.Allow(addr) {
					count++
				}
			}

			mu.Lock()
			total += count
			mu.Unlock()
		}()
	}

	wg.Wait()

	if total != 200*5 {
		t.Fatalf("expected exactly %d allowed, got %d", 200*5, total)
	}
}
