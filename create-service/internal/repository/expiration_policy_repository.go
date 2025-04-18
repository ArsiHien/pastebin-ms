package repository

import (
	"errors"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"gorm.io/gorm"
)

type ExpirationPolicyMySQLRepository struct {
	db *gorm.DB
}

func NewExpirationPolicyMySQLRepository(db *gorm.DB) *ExpirationPolicyMySQLRepository {
	return &ExpirationPolicyMySQLRepository{db: db}
}

func (r *ExpirationPolicyMySQLRepository) FindByPolicyTypeAndDuration(
	policyType paste.ExpirationPolicyType, duration string) (*paste.ExpirationPolicy,
	error) {
	var policy paste.ExpirationPolicy
	err := r.db.Where("policy_type = ? AND duration = ?", policyType, duration).First(&policy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &policy, nil
}

func (r *ExpirationPolicyMySQLRepository) Save(policy *paste.ExpirationPolicy) error {
	if policy.ID != "" {
		return r.db.Save(policy).Error
	}
	return r.db.Create(policy).Error
}
