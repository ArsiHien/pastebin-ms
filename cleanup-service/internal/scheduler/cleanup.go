package scheduler

import (
	"cleanup-service/shared"
	"context"
	"time"

	"cleanup-service/internal/service/cleanup"
)

// CleanupScheduler manages periodic cleanup tasks
type CleanupScheduler struct {
	service *cleanup.Service
	logger  *shared.Logger
}

func NewCleanupScheduler(service *cleanup.Service, logger *shared.Logger) *CleanupScheduler {
	return &CleanupScheduler{
		service: service,
		logger:  logger,
	}
}

func (s *CleanupScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute) // Run every 10 minutes
	defer ticker.Stop()

	s.logger.Info("Starting cleanup scheduler")
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping cleanup scheduler")
			return
		case <-ticker.C:
			s.logger.Info("Running scheduled cleanup")
			count, err := s.service.RunCleanup(ctx)
			if err != nil {
				s.logger.Errorf("Scheduled cleanup failed: %v", err)
				continue
			}
			s.logger.Infof("Scheduled cleanup completed, deleted %d pastes", count)
		}
	}
}
