package paste

import (
	"context"
	"encoding/json"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/metrics"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/shared"
	"go.uber.org/zap"
	"sync"
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
	policyCache          map[string]*paste.ExpirationPolicy // Cache in-memory
	cacheMutex           sync.RWMutex                       // Bảo vệ cache
}

func NewCreatePasteUseCase(pasteRepo paste.Repository,
	expirationPolicyRepo paste.ExpirationPolicyRepository,
	pub paste.EventPublisher) *CreatePasteUseCase {
	return &CreatePasteUseCase{
		PasteRepo:            pasteRepo,
		ExpirationPolicyRepo: expirationPolicyRepo,
		Publisher:            pub,
		policyCache:          make(map[string]*paste.ExpirationPolicy),
	}
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
	cacheKey := string(req.PolicyType) + ":" + normalizedDuration

	// Kiểm tra cache trước
	uc.cacheMutex.RLock()
	expirationPolicy, exists := uc.policyCache[cacheKey]
	uc.cacheMutex.RUnlock()

	if !exists {
		// Cache miss, truy vấn MySQL
		expirationPolicy, err = uc.ExpirationPolicyRepo.FindByPolicyTypeAndDuration(req.PolicyType, normalizedDuration)
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
		}

		// Lưu vào cache
		uc.cacheMutex.Lock()
		uc.policyCache[cacheKey] = expirationPolicy
		uc.cacheMutex.Unlock()
	}

	logger.Info("Found expiration policy", zap.Any("policy", expirationPolicy))
	metrics.CreateRequestDuration.WithLabelValues("expiration_policy").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 5: Chuẩn bị Paste và đưa vào queue
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
	// Publish paste vào queue để lưu MySQL bất đồng bộ
	pasteData, err := json.Marshal(newPaste)
	if err != nil {
		logger.Error("Failed to marshal paste for queue", zap.Error(err))
		return nil, err
	}
	if err := uc.Publisher.PublishPasteSave(ctx, pasteData); err != nil {
		logger.Error("Failed to publish paste to save queue", zap.Error(err))
		return nil, err
	}
	logger.Info("Published paste to save queue", zap.String("url", url))
	metrics.CreateRequestDuration.WithLabelValues("rabbitmq_publish_save").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 6: Publish sự kiện paste.created
	phaseStart = time.Now()
	if err := uc.Publisher.PublishPasteCreated(&newPaste); err != nil {
		logger.Error("Failed to publish paste.created event", zap.Error(err))
		return nil, err
	}
	logger.Info("Published paste.created event", zap.String("url", url))
	metrics.CreateRequestDuration.WithLabelValues("rabbitmq_publish").Observe(time.Since(phaseStart).Seconds())

	return &CreatePasteResponse{URL: newPaste.URL}, nil
}
