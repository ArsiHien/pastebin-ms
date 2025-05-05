package paste

import (
	"context"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/metrics"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/shared"
	"go.uber.org/zap"
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

func (uc *CreatePasteUseCase) Execute(ctx context.Context, req CreatePasteRequest) (
	*CreatePasteResponse, error) {
	logger := zap.L().With(zap.String("requestID", ctx.Value("requestID").(string)))

	// Kiểm tra dữ liệu đầu vào
	if req.Content == "" {
		logger.Error("Empty content")
		return nil, shared.ErrEmptyContent
	}
	if req.PolicyType == paste.TimedExpiration && req.Duration == "" {
		logger.Error("Missing duration for timed expiration")
		return nil, shared.ErrMissingDuration
	}
	normalizedDuration := req.Duration
	if req.PolicyType != paste.TimedExpiration {
		normalizedDuration = ""
	}

	// Giai đoạn 3: Tạo URL ngẫu nhiên
	phaseStart := time.Now()
	url, err := shared.GenerateURL(5)
	if err != nil {
		logger.Error("Failed to generate URL", zap.Error(err))
		return nil, err
	}
	logger.Info("Generated random URL", zap.String("url", url))
	metrics.CreateRequestDuration.WithLabelValues("generate_url").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 4: Tìm hoặc tạo Expiration Policy
	phaseStart = time.Now()
	expirationPolicy, err := uc.ExpirationPolicyRepo.FindByPolicyTypeAndDuration(
		req.PolicyType, normalizedDuration)
	if err != nil {
		logger.Error("Failed to find expiration policy", zap.Error(err))
		return nil, err
	}
	if expirationPolicy == nil {
		expirationPolicy = &paste.ExpirationPolicy{
			Type:     req.PolicyType,
			Duration: normalizedDuration,
		}
		if err := uc.ExpirationPolicyRepo.Save(expirationPolicy); err != nil {
			logger.Error("Failed to save expiration policy", zap.Error(err))
			return nil, err
		}
		logger.Info("Saved new expiration policy", zap.Any("policy", expirationPolicy))
	} else {
		logger.Info("Found existing expiration policy", zap.Any("policy", expirationPolicy))
	}
	metrics.CreateRequestDuration.WithLabelValues("expiration_policy").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 5: Lưu Paste vào MySQL
	phaseStart = time.Now()
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
		logger.Error("Failed to save paste", zap.Error(err))
		return nil, err
	}
	logger.Info("Saved paste to MySQL", zap.String("url", url))
	metrics.CreateRequestDuration.WithLabelValues("mysql_save").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 6: Publish sự kiện
	phaseStart = time.Now()
	if err := uc.Publisher.PublishPasteCreated(&newPaste); err != nil {
		logger.Error("Failed to publish paste.created event", zap.Error(err))
		return nil, err
	}
	logger.Info("Published paste.created event", zap.String("url", url))
	metrics.CreateRequestDuration.WithLabelValues("rabbitmq_publish").Observe(time.Since(phaseStart).Seconds())

	return &CreatePasteResponse{URL: newPaste.URL}, nil
}
