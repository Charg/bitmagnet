package responder

import (
	"context"
	"net/netip"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"
)

// responderHashDiscovery attempts to add info hashes from successful incoming announce_peer and get_peers
// requests to the discovered hashes channel. Sends never block; hashes are dropped when the channel is full.
type responderHashDiscovery struct {
	responder        Responder
	discoveredHashes chan<- dht.DiscoveredHash
}

func (r responderHashDiscovery) Respond(ctx context.Context, msg dht.RecvMsg) (dht.Return, error) {
	ret, err := r.responder.Respond(ctx, msg)
	if err != nil || msg.Msg.A == nil || msg.Msg.A.InfoHash == [20]byte{} {
		return ret, err
	}

	var discovered dht.DiscoveredHash

	switch msg.Msg.Q {
	case dht.QAnnouncePeer:
		discovered = dht.DiscoveredHash{
			InfoHash: msg.Msg.A.InfoHash,
			Source:   dht.QAnnouncePeer,
			From:     msg.From,
		}
		if port := msg.AnnouncePort(); port != 0 {
			discovered.Peer = netip.AddrPortFrom(msg.From.Addr(), port)
		}
	case dht.QGetPeers:
		discovered = dht.DiscoveredHash{
			InfoHash: msg.Msg.A.InfoHash,
			Source:   dht.QGetPeers,
			From:     msg.From,
		}
	default:
		return ret, err
	}

	select {
	case r.discoveredHashes <- discovered:
	default:
	}

	return ret, err
}
