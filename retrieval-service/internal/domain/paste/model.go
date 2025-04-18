package paste

import "time"

type Paste struct {
	URL              string           `json:"url" bson:"url"`
	Content          string           `json:"content" bson:"content"`
	CreatedAt        time.Time        `json:"created_at" bson:"created_at"`
	ExpirationPolicy ExpirationPolicy `json:"expiration_policy" bson:"expiration_policy"`
}

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

type RetrievePasteResponse struct {
	URL           string `json:"url"`
	Content       string `json:"content"`
	RemainingTime string `json:"remaining_time"`
}

type RetrievePolicyResponse struct {
	Policy string `json:"policy"`
}
