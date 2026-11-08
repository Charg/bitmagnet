package dhtcrawler

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/metainfo"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRequester struct {
	fn func(ctx context.Context, addr netip.AddrPort) (metainforequester.Response, error)
}

func (f fakeRequester) Request(
	ctx context.Context,
	_ protocol.ID,
	addr netip.AddrPort,
) (metainforequester.Response, error) {
	return f.fn(ctx, addr)
}

type fakeChecker struct{ err error }

func (f fakeChecker) Check(metainfo.Info) error { return f.err }

type fakeBlocker struct {
	mu      sync.Mutex
	blocked []protocol.ID
}

func (f *fakeBlocker) Filter(_ context.Context, h []protocol.ID) ([]protocol.ID, error) {
	return h, nil
}

func (f *fakeBlocker) Block(_ context.Context, h []protocol.ID, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.blocked = append(f.blocked, h...)

	return nil
}

func (f *fakeBlocker) Flush(context.Context) error { return nil }

func testAddr(port uint16) netip.AddrPort {
	return netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), port)
}

func TestDoRequestMetaInfo(t *testing.T) {
	t.Parallel()

	t.Run("fast second peer wins and slow loser is cancelled", func(t *testing.T) {
		t.Parallel()

		cancelled := make(chan struct{})
		c := &crawler{
			banningChecker:  fakeChecker{},
			blockingManager: &fakeBlocker{},
			metainfoRequester: fakeRequester{
				fn: func(ctx context.Context, a netip.AddrPort) (metainforequester.Response, error) {
					if a.Port() == 1 {
						<-ctx.Done()
						close(cancelled)

						return metainforequester.Response{}, ctx.Err()
					}

					return metainforequester.Response{}, nil
				},
			},
		}

		start := time.Now()
		_, err := c.doRequestMetaInfo(
			context.Background(), protocol.ID{}, []netip.AddrPort{testAddr(1), testAddr(2), testAddr(1)})
		require.NoError(t, err)
		assert.Less(t, time.Since(start), 2*time.Second)

		select {
		case <-cancelled:
		case <-time.After(time.Second):
			t.Fatal("slow attempt was not cancelled")
		}
	})

	t.Run("failure starts next attempt early", func(t *testing.T) {
		t.Parallel()

		c := &crawler{
			banningChecker:  fakeChecker{},
			blockingManager: &fakeBlocker{},
			metainfoRequester: fakeRequester{
				fn: func(_ context.Context, a netip.AddrPort) (metainforequester.Response, error) {
					if a.Port() == 1 {
						return metainforequester.Response{}, errors.New("dead")
					}

					return metainforequester.Response{}, nil
				},
			},
		}

		start := time.Now()
		_, err := c.doRequestMetaInfo(context.Background(), protocol.ID{}, []netip.AddrPort{testAddr(1), testAddr(2)})
		require.NoError(t, err)
		assert.Less(t, time.Since(start), requestMetaInfoStagger)
	})

	t.Run("all fail returns joined error", func(t *testing.T) {
		t.Parallel()

		byPort := map[uint16]error{
			1: errors.New("a"),
			2: errors.New("b"),
			3: errors.New("c"),
			4: errors.New("d"),
		}
		c := &crawler{
			banningChecker:  fakeChecker{},
			blockingManager: &fakeBlocker{},
			metainfoRequester: fakeRequester{
				fn: func(_ context.Context, a netip.AddrPort) (metainforequester.Response, error) {
					return metainforequester.Response{}, byPort[a.Port()]
				},
			},
		}

		_, err := c.doRequestMetaInfo(
			context.Background(),
			protocol.ID{},
			[]netip.AddrPort{testAddr(1), testAddr(2), testAddr(3), testAddr(4)},
		)
		require.Error(t, err)

		for _, e := range byPort {
			require.ErrorIs(t, err, e)
		}
	})

	t.Run("no peers", func(t *testing.T) {
		t.Parallel()

		c := &crawler{}
		_, err := c.doRequestMetaInfo(context.Background(), protocol.ID{}, nil)
		require.Error(t, err)
	})

	t.Run("banned response blocks hash", func(t *testing.T) {
		t.Parallel()

		banErr := errors.New("banned")
		blocker := &fakeBlocker{}
		c := &crawler{
			banningChecker:  fakeChecker{err: banErr},
			blockingManager: blocker,
			metainfoRequester: fakeRequester{
				fn: func(context.Context, netip.AddrPort) (metainforequester.Response, error) {
					return metainforequester.Response{}, nil
				},
			},
		}

		_, err := c.doRequestMetaInfo(context.Background(), protocol.ID{}, []netip.AddrPort{testAddr(1)})
		require.ErrorIs(t, err, banErr)
		assert.Len(t, blocker.blocked, 1)
	})

	t.Run("parent cancellation", func(t *testing.T) {
		t.Parallel()

		c := &crawler{
			banningChecker:  fakeChecker{},
			blockingManager: &fakeBlocker{},
			metainfoRequester: fakeRequester{
				fn: func(ctx context.Context, _ netip.AddrPort) (metainforequester.Response, error) {
					<-ctx.Done()

					return metainforequester.Response{}, ctx.Err()
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()


		_, err := c.doRequestMetaInfo(ctx, protocol.ID{}, []netip.AddrPort{testAddr(1), testAddr(2)})
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
