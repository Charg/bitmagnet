package server

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/errreason"
	"github.com/prometheus/client_golang/prometheus"
)

type prometheusCollector struct {
	queryDuration     *prometheus.HistogramVec
	querySuccessTotal *prometheus.CounterVec
	queryErrorTotal   *prometheus.CounterVec
	queryConcurrency  *prometheus.GaugeVec
}

const (
	labelQuery  = "query"
	labelReason = "reason"
)

var labelNames = []string{labelQuery}

// Reasons for a failed query, in addition to those of errreason.
const (
	reasonKrpcError       = "krpc_error"
	reasonSendFailed      = "send_failed"
	reasonInvalidResponse = "invalid_response"
)

func newPrometheusCollector() prometheusCollector {
	return prometheusCollector{
		queryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "query_duration_seconds",
			Help:      "A histogram of successful DHT query durations in seconds.",
			Buckets:   prometheus.ExponentialBuckets(0.1, 1.5, 5),
		}, labelNames),
		querySuccessTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "query_success_total",
			Help:      "A counter of successful DHT queries.",
		}, labelNames),
		queryErrorTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "query_error_total",
			Help:      "A counter of failed DHT queries, by the reason of failure.",
		}, []string{labelQuery, labelReason}),
		queryConcurrency: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "query_concurrency",
			Help:      "Number of concurrent DHT queries.",
		}, labelNames),
	}
}

type prometheusServerWrapper struct {
	prometheusCollector
	server Server
}

func (s prometheusServerWrapper) start() error {
	return s.server.start()
}

func (s prometheusServerWrapper) stop() {
	s.server.stop()
}

func (s prometheusServerWrapper) Query(
	ctx context.Context,
	addr netip.AddrPort,
	q string,
	args dht.MsgArgs,
) (dht.RecvMsg, error) {
	labels := prometheus.Labels{labelQuery: q}
	s.queryConcurrency.With(labels).Inc()

	start := time.Now()
	res, err := s.server.Query(ctx, addr, q, args)
	s.queryConcurrency.With(labels).Dec()

	if err == nil {
		s.queryDuration.With(labels).Observe(time.Since(start).Seconds())
		s.querySuccessTotal.With(labels).Inc()
	} else {
		s.queryErrorTotal.With(prometheus.Labels{labelQuery: q, labelReason: queryErrorReason(err)}).Inc()
	}

	return res, err
}

func queryErrorReason(err error) string {
	var (
		krpcErr    dht.Error
		krpcErrPtr *dht.Error
	)

	switch {
	case errors.As(err, &krpcErrPtr), errors.As(err, &krpcErr):
		return reasonKrpcError
	case errors.Is(err, ErrInvalidResponse):
		return reasonInvalidResponse
	case errors.Is(err, ErrSendFailed):
		return reasonSendFailed
	default:
		return errreason.Classify(err)
	}
}
