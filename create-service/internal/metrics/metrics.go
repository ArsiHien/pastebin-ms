package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// CreateRequestDuration là histogram để đo thời gian từng giai đoạn trong Create Service
var CreateRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "create_service_request_duration_seconds",
		Help:    "Duration of each phase in Create Service",
		Buckets: prometheus.LinearBuckets(0.001, 0.005, 20),
	},
	[]string{"phase"},
)

func init() {
	prometheus.MustRegister(CreateRequestDuration)
}
