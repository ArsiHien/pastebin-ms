package paste

import (
	"time"
)

// PasteCreatedEvent represents a paste creation event
type PasteCreatedEvent struct {
	URL              string           `json:"url"`
	CreatedAt        time.Time        `json:"created_at"`
	ExpirationPolicy ExpirationPolicy `json:"expiration_policy"`
}

// PasteViewedEvent represents a paste view event
type PasteViewedEvent struct {
	URL      string    `json:"url"`
	ViewedAt time.Time `json:"viewed_at"`
}

// PasteDeletedEvent represents a paste deletion event
type PasteDeletedEvent struct {
	URL       string    `json:"url"`
	DeletedAt time.Time `json:"deleted_at"`
}
