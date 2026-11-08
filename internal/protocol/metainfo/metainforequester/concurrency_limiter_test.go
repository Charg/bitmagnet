package metainforequester

import (
	"context"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingRequester struct {
	current atomic.Int32
	peak    atomic.Int32
}

func (r *countingRequester) Request(context.Context, protocol.ID, netip.AddrPort) (Response, error) {
	n := r.current.Add(1)

	for {
		p := r.peak.Load()
		if n <= p || r.peak.CompareAndSwap(p, n) {
			break
		}
	}

	time.Sleep(10 * time.Millisecond)
	r.current.Add(-1)

	return Response{}, nil
}

func TestConcurrencyLimiter(t *testing.T) {
	t.Parallel()

	inner := &countingRequester{}
	limited := newConcurrencyLimiter(inner, 3)

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := limited.Request(context.Background(), protocol.ID{}, netip.AddrPort{})
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(3), inner.peak.Load())
}

func TestConcurrencyLimiterCancelled(t *testing.T) {
	t.Parallel()

	block := make(chan struct{})
	limited := newConcurrencyLimiter(blockingRequester(block), 1)

	go func() {
		_, _ = limited.Request(context.Background(), protocol.ID{}, netip.AddrPort{})
	}()

	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := limited.Request(ctx, protocol.ID{}, netip.AddrPort{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	close(block)
}

func TestConcurrencyLimiterDisabled(t *testing.T) {
	t.Parallel()

	inner := &countingRequester{}
	assert.Same(t, inner, newConcurrencyLimiter(inner, 0))
}

type blockingRequester chan struct{}

func (b blockingRequester) Request(ctx context.Context, _ protocol.ID, _ netip.AddrPort) (Response, error) {
	select {
	case <-b:
	case <-ctx.Done():
	}

	return Response{}, nil
}
