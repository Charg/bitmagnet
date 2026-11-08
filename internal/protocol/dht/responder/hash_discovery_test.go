package responder

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubResponder struct {
	err error
}

func (s stubResponder) Respond(context.Context, dht.RecvMsg) (dht.Return, error) {
	return dht.Return{}, s.err
}

func hashMsg(q string, hash protocol.ID, port int) dht.RecvMsg {
	return dht.RecvMsg{
		From: netip.MustParseAddrPort("1.2.3.4:6881"),
		Msg:  dht.Msg{Q: q, A: &dht.MsgArgs{ID: protocol.RandomNodeID(), InfoHash: hash, Port: &port}},
	}
}

func TestResponderHashDiscovery(t *testing.T) {
	t.Parallel()

	hash := protocol.RandomNodeID()

	t.Run("announce_peer", func(t *testing.T) {
		t.Parallel()

		ch := make(chan dht.DiscoveredHash, 1)
		r := responderHashDiscovery{responder: stubResponder{}, discoveredHashes: ch}
		_, err := r.Respond(context.Background(), hashMsg(dht.QAnnouncePeer, hash, 5555))
		require.NoError(t, err)
		assert.Equal(t, dht.DiscoveredHash{
			InfoHash: hash,
			Source:   dht.QAnnouncePeer,
			From:     netip.MustParseAddrPort("1.2.3.4:6881"),
			Peer:     netip.MustParseAddrPort("1.2.3.4:5555"),
		}, <-ch)
	})

	t.Run("get_peers", func(t *testing.T) {
		t.Parallel()

		ch := make(chan dht.DiscoveredHash, 1)
		r := responderHashDiscovery{responder: stubResponder{}, discoveredHashes: ch}
		_, err := r.Respond(context.Background(), hashMsg(dht.QGetPeers, hash, 5555))
		require.NoError(t, err)
		assert.Equal(t, dht.DiscoveredHash{
			InfoHash: hash,
			Source:   dht.QGetPeers,
			From:     netip.MustParseAddrPort("1.2.3.4:6881"),
		}, <-ch)
	})

	t.Run("other queries are ignored", func(t *testing.T) {
		t.Parallel()

		ch := make(chan dht.DiscoveredHash, 1)
		r := responderHashDiscovery{responder: stubResponder{}, discoveredHashes: ch}
		_, err := r.Respond(context.Background(), hashMsg(dht.QFindNode, hash, 5555))
		require.NoError(t, err)
		assert.Empty(t, ch)
	})

	t.Run("no emit on error", func(t *testing.T) {
		t.Parallel()

		ch := make(chan dht.DiscoveredHash, 1)
		wantErr := errors.New("invalid token")
		r := responderHashDiscovery{responder: stubResponder{err: wantErr}, discoveredHashes: ch}
		_, err := r.Respond(context.Background(), hashMsg(dht.QAnnouncePeer, hash, 5555))
		require.ErrorIs(t, err, wantErr)
		assert.Empty(t, ch)
	})

	t.Run("does not block when full", func(t *testing.T) {
		t.Parallel()

		ch := make(chan dht.DiscoveredHash, 1)
		r := responderHashDiscovery{responder: stubResponder{}, discoveredHashes: ch}
		_, _ = r.Respond(context.Background(), hashMsg(dht.QGetPeers, hash, 1))
		_, err := r.Respond(context.Background(), hashMsg(dht.QGetPeers, protocol.RandomNodeID(), 1))
		require.NoError(t, err)
		assert.Len(t, ch, 1)
		assert.Equal(t, hash, (<-ch).InfoHash)
	})
}
