package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	PasteProcessingDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "paste_processing_duration_seconds",
			Help:    "Time from paste creation to database insert in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func Init() {
	prometheus.MustRegister(PasteProcessingDuration)
}
