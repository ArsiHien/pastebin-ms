package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// PasteProcessingDuration đo thời gian từ khi tạo paste đến khi lưu vào cơ sở dữ liệu
var PasteProcessingDuration = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "paste_processing_duration_seconds",
		Help:    "Time from paste creation to database insert in seconds.",
		Buckets: prometheus.DefBuckets,
	},
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
	prometheus.MustRegister(PasteProcessingDuration)
	prometheus.MustRegister(RetrievalRequestDuration)
}
