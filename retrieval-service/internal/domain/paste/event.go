package paste

import "time"

type ViewedEvent struct {
	URL      string    `json:"url"`
	ViewedAt time.Time `json:"viewed_at"`
}
