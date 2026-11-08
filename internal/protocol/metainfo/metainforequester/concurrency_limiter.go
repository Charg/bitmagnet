package metainforequester

import (
	"context"
	"net/netip"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"golang.org/x/sync/semaphore"
)

// concurrencyLimiter caps the number of concurrent requests (and therefore outgoing TCP connections).
type concurrencyLimiter struct {
	requester Requester
	sem       *semaphore.Weighted
}

func newConcurrencyLimiter(requester Requester, maxConcurrency uint) Requester {
	if maxConcurrency == 0 {
		return requester
	}

	return concurrencyLimiter{
		requester: requester,
		sem:       semaphore.NewWeighted(int64(maxConcurrency)),
	}
}

func (r concurrencyLimiter) Request(ctx context.Context, infoHash protocol.ID, node netip.AddrPort) (Response, error) {
	if err := r.sem.Acquire(ctx, 1); err != nil {
		return Response{}, err
	}
	defer r.sem.Release(1)

	return r.requester.Request(ctx, infoHash, node)
}
