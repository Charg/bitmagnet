package dhtcrawler

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/metainfo/metainforequester"
)

func (c *crawler) runRequestMetaInfo(ctx context.Context) {
	_ = c.requestMetaInfo.Run(ctx, func(req infoHashWithPeers) {
		mi, reqErr := c.doRequestMetaInfo(ctx, req.infoHash, req.peers)
		if reqErr != nil {
			return
		}
		select {
		case <-ctx.Done():
		case c.persistTorrents.In() <- infoHashWithMetaInfo{
			nodeHasPeersForHash: req.nodeHasPeersForHash,
			metaInfo:            mi.Info,
		}:
		}
	})
}

const (
	// requestMetaInfoMaxConcurrent is the maximum number of peers contacted concurrently for a single hash.
	requestMetaInfoMaxConcurrent = 3
	// requestMetaInfoStagger is the delay before launching the next attempt if none has failed in the meantime.
	requestMetaInfoStagger = 300 * time.Millisecond
)

type metaInfoAttempt struct {
	res metainforequester.Response
	err error
}

// doRequestMetaInfo races requests to the given peers, "happy eyeballs" style, returning the first success.
func (c *crawler) doRequestMetaInfo(
	ctx context.Context,
	hash protocol.ID,
	peers []netip.AddrPort,
) (metainforequester.Response, error) {
	seen := make(map[netip.AddrPort]struct{}, len(peers))
	unique := make([]netip.AddrPort, 0, len(peers))

	for _, p := range peers {
		if _, ok := seen[p]; ok {
			continue
		}

		seen[p] = struct{}{}

		unique = append(unique, p)
	}

	if len(unique) == 0 {
		return metainforequester.Response{}, errors.New("no peers")
	}

	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// buffered so that attempts can always deliver their result without blocking, even after we return
	results := make(chan metaInfoAttempt, len(unique))

	var (
		errs     []error
		next     int
		inFlight int
	)

	timer := time.NewTimer(requestMetaInfoStagger)
	defer timer.Stop()

	launch := func() {
		p := unique[next]
		next++
		inFlight++

		resetTimer(timer, requestMetaInfoStagger)

		go func() {
			res, err := c.metainfoRequester.Request(raceCtx, hash, p)
			results <- metaInfoAttempt{res: res, err: err}
		}()
	}

	launch()

	for inFlight > 0 {
		var timerC <-chan time.Time
		if inFlight < requestMetaInfoMaxConcurrent && next < len(unique) {
			timerC = timer.C
		}

		select {
		case <-raceCtx.Done():
			errs = append(errs, raceCtx.Err())

			return metainforequester.Response{}, errors.Join(errs...)
		case <-timerC:
			launch()
		case r := <-results:
			inFlight--

			if r.err != nil {
				errs = append(errs, r.err)
				// start the next attempt early rather than waiting for the stagger
				if next < len(unique) {
					launch()
				}

				continue
			}

			if banErr := c.banningChecker.Check(r.res.Info); banErr != nil {
				_ = c.blockingManager.Block(ctx, []protocol.ID{hash}, false)

				return metainforequester.Response{}, banErr
			}

			return r.res, nil
		}
	}

	return metainforequester.Response{}, errors.Join(errs...)
}

func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}

	t.Reset(d)
}
