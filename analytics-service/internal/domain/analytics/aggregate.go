package analytics

import (
	"fmt"
	"time"
)

type View struct {
	PasteURL string    `bson:"paste_url"`
	ViewedAt time.Time `bson:"viewed_at"`
}

type Stats struct {
	PasteURL  string `bson:"paste_url"`
	ViewCount int    `bson:"view_count"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	ViewCount int       `json:"viewCount"`
}

type PasteTimeSeriesResponse struct {
	PasteURL   string            `json:"pasteUrl"`
	TotalViews int               `json:"totalViews"`
	TimeSeries []TimeSeriesPoint `json:"timeSeries"`
}

var (
	ErrInvalidPeriod = fmt.Errorf("invalid analytics period")
)
