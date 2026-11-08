package dhtcrawler

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/concurrency"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht/ktable"
	boom "github.com/tylertreat/BoomFilters"
	"go.uber.org/fx"
)

type DiscoveredNodesParams struct {
	fx.In
	Config Config
}

type DiscoveredNodesResult struct {
	fx.Out
	DiscoveredNodes concurrency.BatchingChannel[ktable.Node] `name:"dht_discovered_nodes"`
}

// NewDiscoveredNodes creates the channel for discovered nodes.
// It receives nodes discovered by the crawler, as well as nodes from incoming requests to the DHT server.
// It is provided as a separate service to avoid a circular dependency with the DHT server.
func NewDiscoveredNodes(params DiscoveredNodesParams) DiscoveredNodesResult {
	return DiscoveredNodesResult{
		DiscoveredNodes: concurrency.NewBatchingChannel[ktable.Node](
			int(100*params.Config.ScalingFactor), 10, time.Second/100),
	}
}

// seenNodes is a thread-safe stable bloom filter of recently seen node addresses.
// The routing table only knows about nodes currently in it, so without this filter
// dropped or rejected nodes would be recontacted over and over.
// Being a stable filter, old entries decay so that nodes can eventually be contacted again.
type seenNodes struct {
	mutex sync.Mutex
	bloom *boom.StableBloomFilter
}

func newSeenNodes() *seenNodes {
	return &seenNodes{
		bloom: boom.NewStableBloomFilter(10_000_000, 2, 0.01),
	}
}

// seenNodesOrNil returns a seenNodes filter if enabled, otherwise nil.
func seenNodesOrNil(enabled bool) *seenNodes {
	if !enabled {
		return nil
	}

	return newSeenNodes()
}

// testAndAdd returns true if the address was (probably) already seen, otherwise adds it.
func (s *seenNodes) testAndAdd(addr netip.AddrPort) bool {
	var buf [18]byte

	a := addr.Addr().As16()
	copy(buf[:16], a[:])
	buf[16] = byte(addr.Port() >> 8)
	buf[17] = byte(addr.Port())

	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.bloom.TestAndAdd(buf[:])
}

func (c *crawler) runDiscoveredNodes(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ps := <-c.discoveredNodes.Out():
			addrs := make([]netip.Addr, 0, len(ps))

			m := make(map[netip.Addr]ktable.Node, len(ps))
			for _, p := range ps {
				if _, ok := m[p.Addr().Addr()]; !ok {
					m[p.Addr().Addr()] = p
					addrs = append(addrs, p.Addr().Addr())
				}
			}
			// for any discovered node not already in the routing table, and not recently seen,
			// we will block until it can be sent to either of the discovery pipeline channels.
			// Ping is deliberately not used here: sample_infohashes learns more and reveals BEP51 support.
			unknownAddrs := c.kTable.FilterKnownAddrs(addrs)
			for _, addr := range unknownAddrs {
				p := m[addr]
				if c.seenNodes != nil && c.seenNodes.testAndAdd(p.Addr()) {
					continue
				}

				select {
				case <-ctx.Done():
					return
				case c.nodesForFindNode.In() <- p:
				case c.nodesForSampleInfoHashes.In() <- p:
				}
			}
		}
	}
}
