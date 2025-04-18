package paste

import "time"

type ViewedEvent struct {
	URL      string    `json:"url"`
	ViewedAt time.Time `json:"viewed_at"`
}

type BurnAfterReadPasteViewedEvent struct {
	URL string `json:"url"`
}

type EventPublisher interface {
	PublishPasteViewedEvent(event ViewedEvent) error
	PublishBurnAfterReadPasteViewedEvent(event BurnAfterReadPasteViewedEvent) error
	Close() error
}
