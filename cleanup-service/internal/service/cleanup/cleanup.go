package cleanup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cleanup-service/internal/domain/paste"
	"cleanup-service/internal/eventbus"
	"cleanup-service/internal/repository"
	"cleanup-service/internal/shared"
)

type CleanupService struct {
	mysqlRepo     repository.MySQLPasteRepository
	retrievalRepo repository.RetrievalRepository
	analyticsRepo repository.AnalyticsRepository
	cleanupRepo   repository.CleanupRepository
	publisher     eventbus.EventPublisher
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
	publisher eventbus.EventPublisher,
	consumer eventbus.EventConsumer,
	logger *shared.Logger,
) *CleanupService {
	return &CleanupService{
		mysqlRepo:     mysqlRepo,
		retrievalRepo: retrievalRepo,
		analyticsRepo: analyticsRepo,
		cleanupRepo:   cleanupRepo,
		publisher:     publisher,
		consumer:      consumer,
		logger:        logger,
	}
}

func (s *CleanupService) StartEventConsumer(ctx context.Context) error {
	return s.consumer.Consume(ctx, func(event interface{}) error {
		switch e := event.(type) {
		case paste.PasteCreatedEvent:
			var expireAt time.Time
			isBurnAfterRead := e.ExpirationPolicy.Type == paste.BurnAfterRead
			if e.ExpirationPolicy.Type == paste.TimedExpiration {
				if duration, ok := paste.DurationMap[e.ExpirationPolicy.Duration]; ok {
					expireAt = e.CreatedAt.Add(duration)
				} else {
					return fmt.Errorf("invalid duration: %s", e.ExpirationPolicy.Duration)
				}
			}
			return s.cleanupRepo.AddTask(ctx, e.URL, expireAt, isBurnAfterRead)
		case paste.PasteViewedEvent:
			return s.cleanupRepo.MarkRead(ctx, e.URL)
		default:
			return fmt.Errorf("unknown event type")
		}
	})
}

func (s *CleanupService) RunCleanup(ctx context.Context) (int, error) {
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

func (s *CleanupService) deletePaste(ctx context.Context, url string) error {
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

	// Publish PasteDeletedEvent
	event := paste.PasteDeletedEvent{
		URL:       url,
		DeletedAt: time.Now(),
	}
	if err := s.publisher.PublishPasteDeleted(ctx, event); err != nil {
		s.logger.Errorf("Failed to publish PasteDeletedEvent for %s: %v", url, err)
	}

	s.logger.Infof("Deleted paste %s from all databases", url)
	return nil
}

func (s *CleanupService) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"last_run":       s.lastRun,
		"pastes_deleted": s.pastesDeleted,
	}
}
