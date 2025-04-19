package paste

import (
	"time"
)

type CreatedEvent struct {
	ID               string           `json:"id"`
	URL              string           `json:"url"`
	Content          string           `json:"content"`
	CreatedAt        time.Time        `json:"created_at"`
	ExpirationPolicy ExpirationPolicy `json:"expiration_policy"`
}

type ViewedEvent struct {
	URL      string    `json:"url"`
	ViewedAt time.Time `json:"viewed_at"`
}

type BurnAfterReadPasteViewedEvent struct {
	URL string `json:"url"`
}
