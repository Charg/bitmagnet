package dhtcrawler

import (
	"net/netip"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht/ktable"
	ktable_mocks "github.com/bitmagnet-io/bitmagnet/internal/protocol/dht/ktable/mocks"
	"github.com/stretchr/testify/assert"
	boom "github.com/tylertreat/BoomFilters"
)

func TestRequestForDiscoveredHash(t *testing.T) {
	t.Parallel()

	from := netip.MustParseAddrPort("1.2.3.4:6881")
	peer := netip.MustParseAddrPort("1.2.3.4:5555")
	closestAddr := netip.MustParseAddrPort("5.6.7.8:6881")

	newCrawler := func(t *testing.T) (*crawler, *ktable_mocks.Table) {
		t.Helper()

		table := ktable_mocks.NewTable(t)

		return &crawler{
			kTable:       table,
			ignoreHashes: &ignoreHashes{bloom: boom.NewStableBloomFilter(1000, 2, 0.001)},
		}, table
	}

	t.Run("announce_peer carries peer hint", func(t *testing.T) {
		t.Parallel()

		c, _ := newCrawler(t)
		hash := protocol.RandomNodeID()
		req, ok := c.requestForDiscoveredHash(dht.DiscoveredHash{
			InfoHash: hash, Source: dht.QAnnouncePeer, From: from, Peer: peer,
		})
		assert.True(t, ok)
		assert.Equal(t, nodeHasPeersForHash{infoHash: hash, node: from, peers: []netip.AddrPort{peer}}, req)
	})

	t.Run("get_peers uses closest node", func(t *testing.T) {
		t.Parallel()

		c, table := newCrawler(t)
		hash := protocol.RandomNodeID()
		table.EXPECT().GetClosestNodes(hash).Return([]ktable.Node{
			ktable.NewNode(protocol.RandomNodeID(), closestAddr),
		})

		req, ok := c.requestForDiscoveredHash(
			dht.DiscoveredHash{InfoHash: hash, Source: dht.QGetPeers, From: from},
		)
		assert.True(t, ok)
		assert.Equal(t, nodeHasPeersForHash{infoHash: hash, node: closestAddr}, req)
	})

	t.Run("get_peers falls back to sender", func(t *testing.T) {
		t.Parallel()

		c, table := newCrawler(t)
		hash := protocol.RandomNodeID()
		table.EXPECT().GetClosestNodes(hash).Return(nil)

		req, ok := c.requestForDiscoveredHash(
			dht.DiscoveredHash{InfoHash: hash, Source: dht.QGetPeers, From: from},
		)
		assert.True(t, ok)
		assert.Equal(t, nodeHasPeersForHash{infoHash: hash, node: from}, req)
	})

	t.Run("seen hashes are skipped", func(t *testing.T) {
		t.Parallel()

		c, _ := newCrawler(t)
		hash := protocol.RandomNodeID()
		d := dht.DiscoveredHash{InfoHash: hash, Source: dht.QAnnouncePeer, From: from, Peer: peer}
		_, ok := c.requestForDiscoveredHash(d)
		assert.True(t, ok)
		_, ok = c.requestForDiscoveredHash(d)
		assert.False(t, ok)
	})
}
