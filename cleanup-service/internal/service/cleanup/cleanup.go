package cleanup

import (
	"cleanup-service/internal/domain/paste"
	"cleanup-service/internal/eventbus"
	"cleanup-service/internal/repository"
	"cleanup-service/shared"
	"context"
	"fmt"
	"sync"
	"time"
)

type Service struct {
	mysqlRepo     repository.MySQLPasteRepository
	retrievalRepo repository.RetrievalRepository
	analyticsRepo repository.AnalyticsRepository
	cleanupRepo   repository.CleanupRepository
	consumer      eventbus.EventConsumer
	logger        *shared.Logger

	mu            sync.Mutex
	lastRun       time.Time
	pastesDeleted int
}

func NewCleanupService(
	mysqlRepo repository.MySQLPasteRepository,
	retrievalRepo repository.RetrievalRepository,
	analyticsRepo repository.AnalyticsRepository,
	cleanupRepo repository.CleanupRepository,
	consumer eventbus.EventConsumer,
	logger *shared.Logger,
) *Service {
	return &Service{
		mysqlRepo:     mysqlRepo,
		retrievalRepo: retrievalRepo,
		analyticsRepo: analyticsRepo,
		cleanupRepo:   cleanupRepo,
		consumer:      consumer,
		logger:        logger,
	}
}

// StartEventConsumer handles incoming events
func (s *Service) StartEventConsumer(ctx context.Context) error {
	return s.consumer.Consume(ctx, func(event interface{}) error {
		switch e := event.(type) {
		case paste.CreatedEvent:
			return s.handleCreatedEvent(ctx, e)

		case paste.ViewedEvent:
			return s.cleanupRepo.MarkRead(ctx, e.URL)

		case paste.BurnAfterReadPasteViewedEvent:
			s.logger.Infof("Processing burn after read event for URL: %s", e.URL)

			if err := s.cleanupRepo.MarkRead(ctx, e.URL); err != nil {
				return fmt.Errorf("failed to mark burn after read paste as read: %w", err)
			}

			go func(url string) {
				if err := s.deletePaste(ctx, url, true); err != nil {
					s.logger.Errorf("Failed to delete burn after read paste %s: %v", url, err)
				} else {
					s.logger.Infof("Successfully deleted burn after read paste %s", url)
				}
			}(e.URL)

			return nil

		default:
			return fmt.Errorf("unknown event type")
		}
	})
}

func (s *Service) handleCreatedEvent(ctx context.Context, e paste.CreatedEvent) error {
	var expireAt time.Time
	isBurnAfterRead := false

	switch e.ExpirationPolicy.Type {
	case paste.BurnAfterRead:
		isBurnAfterRead = true
	case paste.TimedExpiration:
		duration, ok := paste.DurationMap[e.ExpirationPolicy.Duration]
		if !ok {
			return fmt.Errorf("invalid duration: %s", e.ExpirationPolicy.Duration)
		}
		expireAt = e.CreatedAt.Add(duration)
	}

	return s.cleanupRepo.AddTask(ctx, e.URL, expireAt, isBurnAfterRead)
}

// RunCleanup deletes expired pastes
func (s *Service) RunCleanup(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	urls, err := s.cleanupRepo.FindExpired(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to find expired pastes: %w", err)
	}

	count := 0
	for _, url := range urls {
		if err := s.deletePaste(ctx, url, false); err != nil {
			s.logger.Errorf("Failed to delete paste %s: %v", url, err)
			continue
		}
		count++
	}

	s.lastRun = time.Now()
	s.pastesDeleted += count
	return count, nil
}

// deletePaste deletes a paste from databases. If burnAfterRead is true, skip analytics deletion.
func (s *Service) deletePaste(ctx context.Context, url string, burnAfterRead bool) error {
	if err := s.mysqlRepo.Delete(ctx, url); err != nil {
		return fmt.Errorf("failed to delete from MySQL: %w", err)
	}

	if err := s.retrievalRepo.Delete(ctx, url); err != nil {
		return fmt.Errorf("failed to delete from MongoDB retrieval: %w", err)
	}

	if !burnAfterRead {
		if err := s.analyticsRepo.Delete(ctx, url); err != nil {
			return fmt.Errorf("failed to delete from MongoDB analytics: %w", err)
		}
	}

	if err := s.cleanupRepo.DeleteTask(ctx, url); err != nil {
		return fmt.Errorf("failed to delete from cleanup_tasks: %w", err)
	}

	s.logger.Infof("Deleted paste %s from all databases", url)
	return nil
}

// GetStatus returns the latest cleanup run metadata
func (s *Service) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"last_run":       s.lastRun,
		"pastes_deleted": s.pastesDeleted,
	}
}
