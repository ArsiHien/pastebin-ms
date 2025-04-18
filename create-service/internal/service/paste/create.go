package paste

import (
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/shared"
	"time"
)

type CreatePasteRequest struct {
	Content    string                     `json:"content"`
	PolicyType paste.ExpirationPolicyType `json:"policyType" bson:"policyType"`
	Duration   string                     `json:"duration,omitempty" bson:"duration,omitempty"`
}

type CreatePasteResponse struct {
	URL string `json:"url"`
}

type CreatePasteUseCase struct {
	PasteRepo            paste.Repository
	ExpirationPolicyRepo paste.ExpirationPolicyRepository
	Publisher            paste.EventPublisher
}

func NewCreatePasteUseCase(pasteRepo paste.Repository,
	expirationPolicyRepo paste.ExpirationPolicyRepository,
	pub paste.EventPublisher) *CreatePasteUseCase {
	return &CreatePasteUseCase{
		PasteRepo:            pasteRepo,
		ExpirationPolicyRepo: expirationPolicyRepo,
		Publisher:            pub}
}

func (uc *CreatePasteUseCase) Execute(req CreatePasteRequest) (
	*CreatePasteResponse, error) {
	if req.Content == "" {
		return nil, shared.ErrEmptyContent
	}
	if req.PolicyType == paste.TimedExpiration && req.Duration == "" {
		return nil, shared.ErrMissingDuration
	}
	normalizedDuration := req.Duration
	if req.PolicyType != paste.TimedExpiration {
		normalizedDuration = ""
	}

	url, err := shared.GenerateURL(5)
	if err != nil {
		return nil, err
	}

	expirationPolicy, err := uc.ExpirationPolicyRepo.FindByPolicyTypeAndDuration(
		req.PolicyType, normalizedDuration)

	if err != nil {
		return nil, err
	}

	if expirationPolicy == nil {
		expirationPolicy = &paste.ExpirationPolicy{
			Type:     req.PolicyType,
			Duration: normalizedDuration,
		}
		if err := uc.ExpirationPolicyRepo.Save(expirationPolicy); err != nil {
			return nil, err
		}
	}

	newPaste := paste.Paste{
		URL:                url,
		Content:            req.Content,
		CreatedAt:          time.Now(),
		ExpirationPolicyID: expirationPolicy.ID,
		ExpirationPolicy: paste.ExpirationPolicy{
			ID:       expirationPolicy.ID,
			Type:     expirationPolicy.Type,
			Duration: expirationPolicy.Duration,
		},
	}

	if err := uc.PasteRepo.Save(&newPaste); err != nil {
		return nil, err
	}

	if err := uc.Publisher.PublishPasteCreated(&newPaste); err != nil {
		return nil, err
	}

	return &CreatePasteResponse{URL: newPaste.URL}, nil
}
