package paste

import (
	"time"
)

// Paste represents a paste entity
type Paste struct {
	URL              string
	Content          string
	CreatedAt        time.Time
	ExpirationPolicy ExpirationPolicy
}

// ExpirationPolicy defines the expiration rules for a paste
type ExpirationPolicy struct {
	Type     string
	Duration string
	IsRead   bool
}

// DurationMap maps expiration durations to time.Duration
var DurationMap = map[string]time.Duration{
	"10minutes": 10 * time.Minute,
	"1hour":     1 * time.Hour,
	"1day":      24 * time.Hour,
	"1week":     7 * 24 * time.Hour,
	"2weeks":    14 * 24 * time.Hour,
	"1month":    30 * 24 * time.Hour,
	"6months":   180 * 24 * time.Hour,
	"1year":     365 * 24 * time.Hour,
}

// Expiration types
const (
	TimedExpiration = "Timed"
	BurnAfterRead   = "BurnAfterRead"
	NeverExpires    = "Never"
)
