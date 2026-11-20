package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SourceUp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "alertwall_source_up",
		Help: "1 if the last poll for a cluster succeeded",
	}, []string{"cluster"})
	PollDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "alertwall_poll_duration_seconds",
		Help:    "Duration of a full poll cycle",
		Buckets: prometheus.DefBuckets,
	})
)