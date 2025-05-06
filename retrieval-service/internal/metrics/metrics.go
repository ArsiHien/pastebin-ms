package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PasteProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "paste_processing_duration_seconds",
			Help:    "Time taken to process paste creation event",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"phase"}, // mongo_save, cache_save, total
	)
)

// RetrievalRequestDuration đo thời gian từng giai đoạn trong Retrieval Service
var RetrievalRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "retrieval_service_request_duration_seconds",
		Help:    "Duration of each phase in Retrieval Service",
		Buckets: prometheus.LinearBuckets(0.001, 0.005, 20),
	},
	[]string{"phase"},
)

func init() {
	prometheus.MustRegister(RetrievalRequestDuration)
}
