package dhtcrawler

import (
	"context"
	"net/netip"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"
	"go.uber.org/fx"
)

type DiscoveredHashesParams struct {
	fx.In
	Config Config
}

type DiscoveredHashesResult struct {
	fx.Out
	DiscoveredHashes chan dht.DiscoveredHash `name:"dht_discovered_hashes"`
}

// NewDiscoveredHashes creates the channel for info hashes passively discovered from incoming DHT queries
// (announce_peer and get_peers). It is provided as a separate service to avoid a circular dependency
// with the DHT server. Producers must send without blocking and drop when the channel is full.
func NewDiscoveredHashes(params DiscoveredHashesParams) DiscoveredHashesResult {
	return DiscoveredHashesResult{
		DiscoveredHashes: make(chan dht.DiscoveredHash, int(100*params.Config.ScalingFactor)),
	}
}

func (c *crawler) runDiscoveredHashes(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case d := <-c.discoveredHashes:
			c.passivelyDiscoveredTotal.WithLabelValues(d.Source).Inc()

			req, ok := c.requestForDiscoveredHash(d)
			if !ok {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case c.infoHashTriage.In() <- req:
			}
		}
	}
}

// requestForDiscoveredHash converts a passively discovered hash to a triage request,
// returning false if the hash has already been seen.
func (c *crawler) requestForDiscoveredHash(d dht.DiscoveredHash) (nodeHasPeersForHash, bool) {
	if c.ignoreHashes.testAndAdd(d.InfoHash) {
		return nodeHasPeersForHash{}, false
	}

	req := nodeHasPeersForHash{infoHash: d.InfoHash, node: d.From}

	if d.Peer.IsValid() {
		req.peers = []netip.AddrPort{d.Peer}
	} else if closest := c.kTable.GetClosestNodes(d.InfoHash); len(closest) > 0 {
		// the querier is looking for peers, so is unlikely to have any; ask a node close to the hash instead.
		req.node = closest[0].Addr()
	}

	return req, true
}
