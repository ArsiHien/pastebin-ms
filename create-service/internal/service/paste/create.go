package paste

import (
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/shared"
	"log"
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
	PasteRepo paste.Repository
	Publisher paste.EventPublisher
}

func NewCreatePasteUseCase(repo paste.Repository,
	pub paste.EventPublisher) *CreatePasteUseCase {
	return &CreatePasteUseCase{PasteRepo: repo, Publisher: pub}
}

func (uc *CreatePasteUseCase) Execute(req CreatePasteRequest) (
	*CreatePasteResponse, error) {
	log.Printf("PolicyType: %v", req.PolicyType)
	if req.Content == "" {
		return nil, shared.ErrEmptyContent
	}
	if req.PolicyType == paste.TimedExpiration && req.Duration == "" {
		return nil, shared.ErrMissingDuration
	}
	url, err := shared.GenerateURL(5)
	if err != nil {
		return nil, err
	}
	newPaste := paste.Paste{
		URL:       url,
		Content:   req.Content,
		CreatedAt: time.Now(),
		ViewCount: 0,
		ExpirationPolicy: paste.ExpirationPolicy{Type: req.PolicyType,
			Duration: req.Duration, IsRead: false},
	}
	if err := uc.PasteRepo.Save(&newPaste); err != nil {
		return nil, err
	}

	if err := uc.Publisher.PublishPasteCreated(&newPaste); err != nil {
		return nil, err
	}

	return &CreatePasteResponse{URL: newPaste.URL}, nil
}
