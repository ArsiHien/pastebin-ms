package paste

import (
	"context"
	"fmt"
	"retrieval-service/internal/cache"
	"retrieval-service/internal/domain/paste"
	"retrieval-service/internal/metrics"
	"retrieval-service/shared"
	"time"
)

// RetrieveService handles paste retrieval operations
type RetrieveService struct {
	repo   paste.Repository
	cache  cache.PasteCache
	pub    paste.EventPublisher
	logger *shared.Logger
}

// NewRetrieveService creates a new paste retrieval service
func NewRetrieveService(
	repo paste.Repository,
	cache cache.PasteCache,
	pub paste.EventPublisher,
	logger *shared.Logger,
) *RetrieveService {
	return &RetrieveService{
		repo:   repo,
		cache:  cache,
		pub:    pub,
		logger: logger,
	}
}

// GetPasteContent retrieves a paste's content by URL
func (s *RetrieveService) GetPasteContent(ctx context.Context, url string) (*paste.RetrievePasteResponse, error) {
	logger := s.logger.With("requestID", ctx.Value("requestID"), "url", url)

	// Giai đoạn 2: Lấy paste
	phaseStart := time.Now()
	p, err := s.fetchPaste(ctx, url)
	if err != nil {
		return nil, err
	}
	metrics.RetrievalRequestDuration.WithLabelValues("fetch_paste").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 3: Kiểm tra trạng thái hết hạn
	phaseStart = time.Now()
	if s.isExpired(p) {
		if err = s.cache.Delete(url); err != nil {
			logger.Errorf("Failed to delete expired paste from cache", "error", err.Error())
		}
		logger.Errorf("Paste expired")
		return nil, shared.ErrPasteExpired
	}
	metrics.RetrievalRequestDuration.WithLabelValues("check_expiration").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 4: Xử lý view
	phaseStart = time.Now()
	if err = s.processView(ctx, p); err != nil {
		logger.Errorf("Failed to process view", "error", err.Error())
		// Continue to return paste even if view processing fails
	}
	metrics.RetrievalRequestDuration.WithLabelValues("process_view").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 5: Tạo response
	phaseStart = time.Now()
	resp := &paste.RetrievePasteResponse{
		URL:           p.URL,
		Content:       p.Content,
		RemainingTime: s.calculateTimeUntilExpiration(p),
	}
	logger.Infof("Prepared response")
	metrics.RetrievalRequestDuration.WithLabelValues("prepare_response").Observe(time.Since(phaseStart).Seconds())

	return resp, nil
}

// GetPastePolicy retrieves a paste's expiration policy
func (s *RetrieveService) GetPastePolicy(ctx context.Context, url string) (string, error) {
	logger := s.logger.With("requestID", ctx.Value("requestID"), "url", url)

	// Giai đoạn 2: Lấy paste
	phaseStart := time.Now()
	p, err := s.fetchPaste(ctx, url)
	if err != nil {
		return "", err
	}
	metrics.RetrievalRequestDuration.WithLabelValues("fetch_paste").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 3: Kiểm tra trạng thái hết hạn
	phaseStart = time.Now()
	if s.isExpired(p) {
		if err = s.cache.Delete(url); err != nil {
			logger.Errorf("Failed to delete expired paste from cache", "error", err.Error())
		}
		logger.Errorf("Paste expired")
		return "", shared.ErrPasteExpired
	}
	metrics.RetrievalRequestDuration.WithLabelValues("check_expiration").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 4: Tạo response
	phaseStart = time.Now()
	resp := s.calculateTimeUntilExpiration(p)
	logger.Infof("Prepared response")
	metrics.RetrievalRequestDuration.WithLabelValues("prepare_response").Observe(time.Since(phaseStart).Seconds())

	return resp, nil
}

// fetchPaste retrieves a paste from cache or repository
func (s *RetrieveService) fetchPaste(ctx context.Context, url string) (*paste.Paste, error) {
	logger := s.logger.With("requestID", ctx.Value("requestID"), "url", url)

	// Giai đoạn 2.1: Kiểm tra cache
	phaseStart := time.Now()
	p, err := s.cache.Get(url)
	if err != nil {
		logger.Errorf("Cache error", "error", err.Error())
	}
	if p != nil {
		logger.Infof("Cache hit")
		metrics.RetrievalRequestDuration.WithLabelValues("redis_check").Observe(time.Since(phaseStart).Seconds())
		return p, nil
	}
	logger.Infof("Cache miss")
	metrics.RetrievalRequestDuration.WithLabelValues("redis_check").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 2.2: Truy vấn MongoDB
	phaseStart = time.Now()
	p, err = s.repo.FindByURL(url)
	if err != nil {
		logger.Errorf("Failed to find paste in MongoDB", "error", err.Error())
		return nil, fmt.Errorf("failed to find paste: %w", err)
	}
	if p == nil {
		logger.Errorf("Paste not found in MongoDB")
		return nil, shared.ErrPasteNotFound
	}
	logger.Infof("Retrieved paste from MongoDB")
	metrics.RetrievalRequestDuration.WithLabelValues("mongodb_query").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 2.3: Lưu vào cache
	phaseStart = time.Now()
	if p.ExpirationPolicy.Type == paste.TimedExpiration {
		if err = s.cache.Set(p); err != nil {
			logger.Errorf("Failed to cache paste", "error", err.Error())
			// Continue even if caching fails
		} else {
			logger.Infof("Cached paste")
		}
	}
	metrics.RetrievalRequestDuration.WithLabelValues("redis_set").Observe(time.Since(phaseStart).Seconds())

	return p, nil
}

// isExpired checks if a paste has expired
func (s *RetrieveService) isExpired(p *paste.Paste) bool {
	switch p.ExpirationPolicy.Type {
	case paste.TimedExpiration:
		duration, ok := shared.DurationMap[p.ExpirationPolicy.Duration]
		if !ok {
			return false
		}
		return time.Now().After(p.CreatedAt.Add(duration))
	case paste.BurnAfterReadExpiration:
		return p.ExpirationPolicy.IsRead
	case paste.NeverExpiration:
		return false
	default:
		return false
	}
}

// processView handles the view event for a paste
func (s *RetrieveService) processView(ctx context.Context, p *paste.Paste) error {
	logger := s.logger.With("requestID", ctx.Value("requestID"), "url", p.URL)

	// Giai đoạn 4.1: Xử lý burn after read
	phaseStart := time.Now()
	if p.ExpirationPolicy.Type == paste.BurnAfterReadExpiration && !p.ExpirationPolicy.IsRead {
		if err := s.repo.MarkAsRead(p.URL); err != nil {
			logger.Errorf("Failed to mark paste as read", "error", err.Error())
			return err
		}
		p.ExpirationPolicy.IsRead = true
		if err := s.pub.PublishBurnAfterReadPasteViewedEvent(paste.BurnAfterReadPasteViewedEvent{
			URL: p.URL,
		}); err != nil {
			logger.Errorf("Failed to publish burn_after_read event", "error", err.Error())
			return err
		}
		logger.Infof("Published burn_after_read event")
		metrics.RetrievalRequestDuration.WithLabelValues("rabbitmq_publish_burn").Observe(time.Since(phaseStart).Seconds())
		return nil
	}
	metrics.RetrievalRequestDuration.WithLabelValues("rabbitmq_publish_burn").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 4.2: Publish regular view event
	phaseStart = time.Now()
	if err := s.pub.PublishPasteViewedEvent(paste.ViewedEvent{
		URL:      p.URL,
		ViewedAt: time.Now(),
	}); err != nil {
		logger.Errorf("Failed to publish paste_viewed event", "error", err.Error())
		return err
	}
	logger.Infof("Published paste_viewed event")
	metrics.RetrievalRequestDuration.WithLabelValues("rabbitmq_publish_viewed").Observe(time.Since(phaseStart).Seconds())

	return nil
}

// calculateTimeUntilExpiration returns a human-readable string for remaining time
func (s *RetrieveService) calculateTimeUntilExpiration(p *paste.Paste) string {
	switch p.ExpirationPolicy.Type {
	case paste.TimedExpiration:
		duration, ok := shared.DurationMap[p.ExpirationPolicy.Duration]
		if !ok {
			return "unknown"
		}

		expirationTime := p.CreatedAt.Add(duration)
		remaining := time.Until(expirationTime)
		if remaining <= 0 {
			return "expired"
		}

		days := int(remaining.Hours() / 24)
		hours := int(remaining.Hours()) % 24
		minutes := int(remaining.Minutes()) % 60

		if days > 0 {
			return fmt.Sprintf("%d days, %d hours", days, hours)
		}
		if hours > 0 {
			return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
		}
		return fmt.Sprintf("%d minutes", minutes)

	case paste.BurnAfterReadExpiration:
		if p.ExpirationPolicy.IsRead {
			return "expired"
		}
		return "after reading"

	case paste.NeverExpiration:
		return "never"

	default:
		return "unknown"
	}
}
