package paste

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type ExpirationPolicyType string

const (
	TimedExpiration         ExpirationPolicyType = "TIMED"
	NeverExpiration         ExpirationPolicyType = "NEVER"
	BurnAfterReadExpiration ExpirationPolicyType = "BURN_AFTER_READ"
)

type ExpirationPolicy struct {
	ID       string               `gorm:"primaryKey;type:char(36)"`
	Type     ExpirationPolicyType `gorm:"column:policy_type;type:varchar(20);not null" json:"type" bson:"type"`
	Duration string               `gorm:"type:varchar(50)" json:"duration,omitempty" bson:"duration,omitempty"`
}

func (ep *ExpirationPolicy) BeforeCreate(*gorm.DB) error {
	if ep.ID == "" {
		ep.ID = uuid.New().String()
	}
	return nil
}

type Paste struct {
	ID                 string           `gorm:"primaryKey;type:char(36)" json:"id" bson:"id"`
	URL                string           `gorm:"type:varchar(255);unique;not null" json:"url" bson:"url"`
	Content            string           `gorm:"type:text;not null" json:"content" bson:"content"`
	CreatedAt          time.Time        `gorm:"autoCreateTime" json:"created_at" bson:"created_at"`
	ExpirationPolicyID string           `gorm:"type:char(36);not null" json:"expiration_policy_id" bson:"expiration_policy_id"`
	ExpirationPolicy   ExpirationPolicy `gorm:"foreignKey:ExpirationPolicyID;references:ID" json:"expiration_policy" bson:"-"`
}

func (p *Paste) BeforeCreate(*gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}
