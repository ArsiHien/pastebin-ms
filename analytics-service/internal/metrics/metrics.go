package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

var (
	PasteEventDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "paste_event_duration_seconds",
			Help:    "Histogram of durations from 'viewedAt' to saving the paste in the DB.",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	prometheus.MustRegister(PasteEventDuration)
}

func ExposeMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil)) // Exposing metrics on port 8080
	}()
}
