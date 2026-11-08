package responder

import (
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/concurrency"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht/ktable"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
)

type Params struct {
	fx.In
	Config           Config
	KTable           ktable.Table
	DiscoveredNodes  concurrency.BatchingChannel[ktable.Node] `name:"dht_discovered_nodes"`
	DiscoveredHashes chan dht.DiscoveredHash                  `name:"dht_discovered_hashes"`
	Logger           *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Responder         Responder
	QueryDuration     prometheus.Collector `group:"prometheus_collectors"`
	QuerySuccessTotal prometheus.Collector `group:"prometheus_collectors"`
	QueryErrorTotal   prometheus.Collector `group:"prometheus_collectors"`
	QueryConcurrency  prometheus.Collector `group:"prometheus_collectors"`
}

const (
	namespace = "bitmagnet"
	subsystem = "dht_responder"
)

func New(p Params) Result {
	collector := newPrometheusCollector(responderLimiter{
		responder: responder{
			nodeID:                   p.KTable.Origin(),
			kTable:                   p.KTable,
			tokenSecret:              protocol.RandomNodeID().Bytes(),
			sampleInfoHashesInterval: 10,
		},
		limiter: NewLimiter(
			rate.Limit(p.Config.MaxRequestsPerSecond),
			int(p.Config.MaxRequestsBurst),
			rate.Every(time.Second),
			10,
			100_000,
			time.Second*20,
		),
	})

	return Result{
		Responder: responderHashDiscovery{
			discoveredHashes: p.DiscoveredHashes,
			responder: responderNodeDiscovery{
				responder: responderLogger{
					responder: collector,
					logger: p.Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
						return zapcore.NewSamplerWithOptions(core, time.Minute, 10, 0)
					})).Named(subsystem),
				},
				discoveredNodes: p.DiscoveredNodes.In(),
			},
		},
		QueryDuration:     collector.queryDuration,
		QuerySuccessTotal: collector.querySuccessTotal,
		QueryErrorTotal:   collector.queryErrorTotal,
		QueryConcurrency:  collector.queryConcurrency,
	}
}
