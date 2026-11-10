package metainforequester

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/errreason"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCollector(timeout time.Duration) *prometheusCollector {
	return newPrometheusCollector(requester{
		clientID: protocol.RandomPeerID(),
		timeout:  timeout,
		dialer:   &net.Dialer{Timeout: timeout, KeepAlive: -1},
	})
}

func TestCollectorLabelsErrorsByStageAndReason(t *testing.T) {
	t.Parallel()

	t.Run("refused dial", func(t *testing.T) {
		t.Parallel()

		ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		addr := netip.MustParseAddrPort(ln.Addr().String())
		require.NoError(t, ln.Close())

		c := newTestCollector(time.Second)
		_, reqErr := c.Request(context.Background(), protocol.RandomNodeID(), addr)
		require.Error(t, reqErr)

		assert.InDelta(
			t,
			1,
			testutil.ToFloat64(c.requestErrorTotal.WithLabelValues(StageDial, errreason.Refused)),
			0,
		)
	})

	t.Run("peer never responds to the handshake", func(t *testing.T) {
		t.Parallel()

		ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		t.Cleanup(func() { _ = ln.Close() })

		go func() {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}

			t.Cleanup(func() { _ = conn.Close() })
		}()

		c := newTestCollector(200 * time.Millisecond)
		_, reqErr := c.Request(context.Background(), protocol.RandomNodeID(),
			netip.MustParseAddrPort(ln.Addr().String()))
		require.Error(t, reqErr)

		// the connection is closed under the pending read when the request times out,
		// which should be reported as a timeout rather than a closed connection:
		count := testutil.ToFloat64(c.requestErrorTotal.WithLabelValues(StageBtHandshake, errreason.Timeout))
		assert.InDelta(t, 1, count, 0, reqErr.Error())
	})
}
