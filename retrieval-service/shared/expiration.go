package shared

import "time"

// DurationMap maps expiration durations
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
