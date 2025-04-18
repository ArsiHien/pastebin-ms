package paste

type ExpirationPolicyRepository interface {
	FindByPolicyTypeAndDuration(policyType ExpirationPolicyType,
		duration string) (*ExpirationPolicy, error)
	Save(policy *ExpirationPolicy) error
}

type Repository interface {
	Save(paste *Paste) error
}
