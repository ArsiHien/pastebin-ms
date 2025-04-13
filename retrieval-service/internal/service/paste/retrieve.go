package paste

import (
	"fmt"
	"retrieval-service/shared"
	"strings"
	"time"

	"retrieval-service/internal/cache" // Import mới
	"retrieval-service/internal/domain/paste"
)

// DurationMap maps expiration durations
var DurationMap = map[string]time.Duration{
	"10minutes": 10 * time.Minute,
	"1hour":     1 * time.Hour,
	"1day":      24 * time.Hour,
	"1week":     7 * 24 * time.Hour,
	"2weeks":    14 * 24 * time.Hour,
	"1month":    30 * 24 * time.Hour,
	"6months":   180 * 24 * time.Hour,
	"1year":     365 * time.Hour,
}

type RetrieveService struct {
	repo   paste.PasteRepository
	cache  cache.PasteCache // Sử dụng cache.PasteCache
	pub    shared.EventPublisher
	logger *shared.Logger
}

func NewRetrieveService(
	repo paste.PasteRepository,
	cache cache.PasteCache, // Cập nhật
	pub shared.EventPublisher,
) *RetrieveService {
	return &RetrieveService{
		repo:   repo,
		cache:  cache,
		pub:    pub,
		logger: shared.NewLogger(),
	}
}

func (s *RetrieveService) GetPaste(url string) (*paste.RetrievePasteResponse, error) {
	// Check cache
	p, err := s.cache.Get(url)
	if err != nil {
		s.logger.Errorf("Cache error for URL %s: %v", url, err)
	}

	// If not in cache, fetch from repository
	if p == nil {
		p, err = s.repo.FindByURL(url)
		if err != nil {
			return nil, fmt.Errorf("failed to find paste: %w", err)
		}
		if p == nil {
			return nil, shared.ErrPasteNotFound
		}
		// Cache the paste
		if err = s.cache.Set(p); err != nil {
			s.logger.Errorf("Failed to cache paste %s: %v", url, err)
		}
	}

	// Check expiration
	if s.isExpired(p) {
		// Delete from database and cache
		if err = s.repo.Delete(url); err != nil {
			s.logger.Errorf("Failed to delete expired paste %s: %v", url, err)
		}
		if err = s.cache.Delete(url); err != nil {
			s.logger.Errorf("Failed to delete expired paste from cache %s: %v", url, err)
		}
		return nil, shared.ErrPasteExpired
	}

	// Process view
	if err = s.processView(p); err != nil {
		s.logger.Errorf("Failed to process view for paste %s: %v", url, err)
		// Continue to return paste even if view processing fails
	}

	// Calculate time until expiration
	timeUntilExp := s.calculateTimeUntilExpiration(p)

	return &paste.RetrievePasteResponse{
		URL:                 p.URL,
		Content:             p.Content,
		TimeUntilExpiration: timeUntilExp,
	}, nil
}

// ... giữ nguyên các hàm isExpired, processView, calculateTimeUntilExpiration ...
