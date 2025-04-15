package analytics

import (
	"context"
	"fmt"
	"time"

	"analytics-service/internal/domain/analytics"
	"analytics-service/internal/eventbus"
	"analytics-service/internal/repository"
	"analytics-service/internal/shared"
)

const (
	Hourly  = "hourly"
	Weekly  = "weekly"
	Monthly = "monthly"
)

var (
	ErrInvalidPeriod = fmt.Errorf("invalid analytics period")
)

type AnalyticsService struct {
	repo     repository.AnalyticsRepository
	consumer eventbus.EventConsumer
	logger   *shared.Logger
}

func NewAnalyticsService(
	repo repository.AnalyticsRepository,
	consumer eventbus.EventConsumer,
	logger *shared.Logger,
) *AnalyticsService {
	return &AnalyticsService{
		repo:     repo,
		consumer: consumer,
		logger:   logger,
	}
}

func (s *AnalyticsService) StartConsumer(ctx context.Context) error {
	return s.consumer.Consume(ctx, func(event analytics.PasteViewedEvent) error {
		// Save view to paste_views
		view := &analytics.View{
			PasteURL: event.URL,
			ViewedAt: event.ViewedAt,
		}
		if err := s.repo.SaveView(ctx, view); err != nil {
			s.logger.Errorf("Failed to save view for %s: %v", event.URL, err)
			return err
		}

		// Increment view count in paste_stats
		if err := s.repo.IncrementViewCount(ctx, event.URL); err != nil {
			s.logger.Errorf("Failed to increment view count for %s: %v", event.URL, err)
			return err
		}

		return nil
	})
}

func (s *AnalyticsService) GetAnalytics(ctx context.Context, pasteURL string, period string) (*analytics.AnalyticsResponse, error) {
	points, totalViews, err := s.repo.GetAnalytics(ctx, pasteURL, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	return &analytics.Response{
		PasteURL:   pasteURL,
		TotalViews: totalViews,
		TimeSeries: points,
	}, nil
}

func (s *AnalyticsService) GetPastesStats(ctx context.Context) (map[string]int, error) {
	return s.repo.GetPastesStats(ctx)
}
