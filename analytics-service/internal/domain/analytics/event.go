package analytics

import (
	"time"
)

type PasteViewedEvent struct {
	URL      string    `json:"url"`
	ViewedAt time.Time `json:"viewed_at"`
}
