package dhtcrawler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht/ktable"
)

const (
	sampleInfoHashesErrorBackoff = time.Hour
	maxSampleInfoHashesInterval  = 6 * 60 * 60
	earlySampleInfoHashesRequery = 60
	shortSampleInfoHashesLimit   = 300
)

// sampleInfoHashesInterval returns the number of seconds to wait before sampling a node again.
// Nodes such as libtorrent precompute their samples and serve the same ones for the whole advertised interval,
// so re-querying early is wasted unless the node has many more hashes than it returned
// and advertises a short interval (meaning it likely rotates its samples quickly).
// Otherwise the advertised interval is honoured, capped to a sane maximum.
func sampleInfoHashesInterval(interval, num, numSamples int) int {
	if interval <= shortSampleInfoHashesLimit && num > 2*numSamples {
		return min(interval, earlySampleInfoHashesRequery)
	}

	return min(interval, maxSampleInfoHashesInterval)
}

func (c *crawler) getNodesForSampleInfoHashes(ctx context.Context) {
	for {
		peers := c.kTable.GetNodesForSampleInfoHashes(60)
		for _, p := range peers {
			select {
			case <-ctx.Done():
				return
			case c.nodesForSampleInfoHashes.In() <- p:
				continue
			}
		}

		<-time.After(time.Second)
	}
}

func (c *crawler) runSampleInfoHashes(ctx context.Context) {
	_ = c.nodesForSampleInfoHashes.Run(ctx, func(n ktable.Node) {
		if !n.IsSampleInfoHashesCandidate() {
			return
		}

		res, err := c.client.SampleInfoHashes(ctx, n.Addr(), c.soughtNodeID.Get())
		if err != nil {
			var krpcErr dht.Error
			if errors.As(err, &krpcErr) {
				// the node responded, so it is alive and still useful for routing.
				options := []ktable.NodeOption{ktable.NodeResponded()}
				if krpcErr.Code == dht.ErrorCodeMethodUnknown {
					// the node doesn't support BEP51, so it will never be sampled again.
					options = append(options, ktable.NodeBep51Support(false))
				} else {
					options = append(options, ktable.NodeSampleInfoHashesRes(
						0, 0, time.Now().Add(sampleInfoHashesErrorBackoff),
					))
				}

				c.kTable.BatchCommand(ktable.PutNode{ID: n.ID(), Addr: n.Addr(), Options: options})

				return
			}

			c.kTable.BatchCommand(
				ktable.DropNode{ID: n.ID(), Reason: fmt.Errorf("sample_infohashes failed: %w", err)},
			)

			return
		}

		var discoveredHashes []nodeHasPeersForHash

		for _, s := range res.Samples {
			if !c.ignoreHashes.testAndAdd(s) {
				discoveredHashes = append(discoveredHashes, nodeHasPeersForHash{
					infoHash: s,
					node:     n.Addr(),
				})
			}
		}

		for _, h := range discoveredHashes {
			select {
			case <-ctx.Done():
				return
			case c.infoHashTriage.In() <- h:
				continue
			}
		}

		interval := sampleInfoHashesInterval(res.Interval, res.Num, len(res.Samples))

		c.kTable.BatchCommand(ktable.PutNode{ID: n.ID(), Addr: n.Addr(), Options: []ktable.NodeOption{
			ktable.NodeResponded(),
			ktable.NodeBep51Support(true),
			ktable.NodeSampleInfoHashesRes(
				len(discoveredHashes),
				res.Num,
				time.Now().Add(time.Duration(interval)*time.Second),
			),
		}})

		if len(res.Nodes) > 0 {
			// block on the channel for up to a second trying to add sampled nodes to the discoveredNodes
			// channel
			go func() {
				timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()

				for _, n := range res.Nodes {
					select {
					case <-timeoutCtx.Done():
						return
					case c.discoveredNodes.In() <- ktable.NewNode(n.ID, n.Addr):
						continue
					}
				}
			}()
		}
	})
}
