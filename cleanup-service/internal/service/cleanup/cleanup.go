package cleanup

import (
	"cleanup-service/shared"
	"context"
	"fmt"
	"sync"
	"time"

	"cleanup-service/internal/domain/paste"
	"cleanup-service/internal/eventbus"
	"cleanup-service/internal/repository"
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

func (s *Service) StartEventConsumer(ctx context.Context) error {
	return s.consumer.Consume(ctx, func(event interface{}) error {
		switch e := event.(type) {
		case paste.CreatedEvent:
			var expireAt time.Time
			isBurnAfterRead := false

			if e.ExpirationPolicy.Type == paste.BurnAfterRead {
				isBurnAfterRead = true
			} else if e.ExpirationPolicy.Type == paste.TimedExpiration {
				if duration, ok := paste.DurationMap[e.ExpirationPolicy.Duration]; ok {
					expireAt = e.CreatedAt.Add(duration)
				} else {
					return fmt.Errorf("invalid duration: %s", e.ExpirationPolicy.Duration)
				}
			}

			return s.cleanupRepo.AddTask(ctx, e.URL, expireAt, isBurnAfterRead)

		case paste.ViewedEvent:
			return s.cleanupRepo.MarkRead(ctx, e.URL)

		case paste.BurnAfterReadPasteViewedEvent:
			// Handle burn after read event by marking as read and scheduling immediate cleanup
			s.logger.Infof("Processing burn after read event for URL: %s", e.URL)
			if err := s.cleanupRepo.MarkRead(ctx, e.URL); err != nil {
				return fmt.Errorf("failed to mark burn after read paste as read: %w", err)
			}

			// Optionally trigger immediate cleanup for this specific paste
			go func(url string) {
				if err := s.deletePaste(context.Background(), url); err != nil {
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
func (s *Service) RunCleanup(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get expired paste URLs
	urls, err := s.cleanupRepo.FindExpired(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to find expired pastes: %w", err)
	}

	count := 0
	for _, url := range urls {
		if err := s.deletePaste(ctx, url); err != nil {
			s.logger.Errorf("Failed to delete paste %s: %v", url, err)
			continue
		}
		count++
	}

	s.lastRun = time.Now()
	s.pastesDeleted += count
	return count, nil
}

func (s *Service) deletePaste(ctx context.Context, url string) error {
	// Delete from MySQL
	if err := s.mysqlRepo.Delete(ctx, url); err != nil {
		return fmt.Errorf("failed to delete from MySQL: %w", err)
	}

	// Delete from MongoDB (retrieval)
	if err := s.retrievalRepo.Delete(ctx, url); err != nil {
		return fmt.Errorf("failed to delete from MongoDB retrieval: %w", err)
	}

	// Delete from MongoDB (analytics)
	if err := s.analyticsRepo.Delete(ctx, url); err != nil {
		return fmt.Errorf("failed to delete from MongoDB analytics: %w", err)
	}

	// Delete from cleanup_tasks
	if err := s.cleanupRepo.DeleteTask(ctx, url); err != nil {
		return fmt.Errorf("failed to delete from cleanup_tasks: %w", err)
	}

	s.logger.Infof("Deleted paste %s from all databases", url)
	return nil
}

func (s *Service) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"last_run":       s.lastRun,
		"pastes_deleted": s.pastesDeleted,
	}
}
