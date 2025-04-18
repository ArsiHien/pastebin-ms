package paste

import (
	"fmt"
	"retrieval-service/internal/cache"
	"retrieval-service/internal/domain/paste"
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
) *RetrieveService {
	return &RetrieveService{
		repo:   repo,
		cache:  cache,
		pub:    pub,
		logger: shared.NewLogger(),
	}
}

// GetPasteContent retrieves a paste's content by URL
func (s *RetrieveService) GetPasteContent(url string) (*paste.RetrievePasteResponse, error) {
	p, err := s.fetchPaste(url)
	if err != nil {
		return nil, err
	}

	if s.isExpired(p) {
		if err = s.cache.Delete(url); err != nil {
			s.logger.Errorf("Failed to delete expired paste from cache %s: %v", url, err)
		}
		return nil, shared.ErrPasteExpired
	}

	if err = s.processView(p); err != nil {
		s.logger.Errorf("Failed to process view for paste %s: %v", url, err)
		// Continue to return paste even if view processing fails
	}

	return &paste.RetrievePasteResponse{
		URL:           p.URL,
		Content:       p.Content,
		RemainingTime: s.calculateTimeUntilExpiration(p),
	}, nil
}

// GetPastePolicy retrieves a paste's expiration policy
func (s *RetrieveService) GetPastePolicy(url string) (string, error) {
	p, err := s.fetchPaste(url)
	if err != nil {
		return "", err
	}

	if s.isExpired(p) {
		if err = s.cache.Delete(url); err != nil {
			s.logger.Errorf("Failed to delete expired paste from cache %s: %v", url, err)
		}
		return "", shared.ErrPasteExpired
	}

	return s.calculateTimeUntilExpiration(p), nil
}

// fetchPaste retrieves a paste from cache or repository
func (s *RetrieveService) fetchPaste(url string) (*paste.Paste, error) {
	// Check cache first
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

		// Cache the paste if using timed expiration
		if p.ExpirationPolicy.Type == paste.TimedExpiration {
			if err = s.cache.Set(p); err != nil {
				s.logger.Errorf("Failed to cache paste %s: %v", url, err)
				// Continue even if caching fails
			}
		}
	}

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
func (s *RetrieveService) processView(p *paste.Paste) error {
	// If burn after read, mark as read
	if p.ExpirationPolicy.Type == paste.BurnAfterReadExpiration && !p.ExpirationPolicy.IsRead {
		if err := s.repo.MarkAsRead(p.URL); err != nil {
			return err
		}

		p.ExpirationPolicy.IsRead = true
		return s.pub.PublishBurnAfterReadPasteViewedEvent(paste.BurnAfterReadPasteViewedEvent{
			URL: p.URL,
		})
	}

	// Publish regular view event
	return s.pub.PublishPasteViewedEvent(paste.ViewedEvent{
		URL:      p.URL,
		ViewedAt: time.Now(),
	})
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
