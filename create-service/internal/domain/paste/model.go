package paste

import "time"

type ExpirationPolicyType string

const (
	TimedExpiration         ExpirationPolicyType = "TIMED"
	NeverExpiration         ExpirationPolicyType = "NEVER"
	BurnAfterReadExpiration ExpirationPolicyType = "BURN_AFTER_READ"
)

type ExpirationPolicy struct {
	Type     ExpirationPolicyType `json:"type" bson:"type"`
	Duration string               `json:"duration,omitempty" bson:"duration,omitempty"`
	IsRead   bool                 `json:"is_read,omitempty" bson:"is_read,omitempty"`
}

type Paste struct {
	URL              string           `json:"url" bson:"url"`
	Content          string           `json:"content" bson:"content"`
	CreatedAt        time.Time        `json:"created_at" bson:"created_at"`
	ViewCount        int              `json:"view_count" bson:"view_count"`
	ExpirationPolicy ExpirationPolicy `json:"expiration_policy" bson:"expiration_policy"`
}
