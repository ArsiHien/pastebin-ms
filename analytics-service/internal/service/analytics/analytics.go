package analytics

import (
	"analytics-service/internal/domain/analytics"
	"analytics-service/internal/eventbus"
	"analytics-service/shared"
	"context"
	"fmt"
	"time"
)

const (
	Hourly  = "hourly"
	Weekly  = "weekly"
	Monthly = "monthly"
)

type Service struct {
	repo     analytics.Repository
	consumer eventbus.EventConsumer
	logger   *shared.Logger
}

func NewAnalyticsService(repo analytics.Repository, consumer eventbus.EventConsumer, logger *shared.Logger) *Service {
	return &Service{repo: repo, consumer: consumer, logger: logger}
}

func (s *Service) StartConsumer(ctx context.Context) error {
	return s.consumer.Consume(ctx, func(event analytics.PasteViewedEvent) error {
		view := &analytics.View{
			PasteURL: event.URL,
			ViewedAt: event.ViewedAt,
		}

		if err := s.repo.SaveView(ctx, view); err != nil {
			s.logger.Errorf("Failed to save view for %s: %v", event.URL, err)
			return err
		}

		if err := s.repo.IncrementViewCount(ctx, event.URL); err != nil {
			s.logger.Errorf("Failed to increment view count for %s: %v", event.URL, err)
			return err
		}

		s.logger.Infof("Processed view event for paste: %s", event.URL)
		return nil
	})
}

func (s *Service) GetAnalytics(ctx context.Context, pasteURL string, period string) (*analytics.PasteTimeSeriesResponse, error) {
	views, err := s.repo.GetAnalytics(ctx, pasteURL, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	points := generateTimeSeries(views, period)

	return &analytics.PasteTimeSeriesResponse{
		PasteURL:   pasteURL,
		TotalViews: len(views),
		TimeSeries: points,
	}, nil
}

func (s *Service) GetPasteStats(ctx context.Context, pasteURL string) (int, error) {
	return s.repo.GetViewCount(ctx, pasteURL)
}

func generateTimeSeries(views []analytics.View, period string) []analytics.TimeSeriesPoint {
	now := time.Now().In(time.Local)
	viewsByTimeSlot := make(map[time.Time]int)
	var timeSlots []time.Time

	switch period {
	case Hourly:
		startTime := now.Add(-1 * time.Hour)
		for _, view := range views {
			localTime := view.ViewedAt.In(time.Local)
			truncatedMinutes := (localTime.Minute() / 10) * 10
			slot := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), localTime.Hour(), truncatedMinutes, 0, 0, time.Local)
			viewsByTimeSlot[slot]++
		}

		for i := 0; i < 6; i++ {
			minutes := (startTime.Minute() / 10) * 10
			slot := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), startTime.Hour(), minutes+i*10, 0, 0, time.Local)
			timeSlots = append(timeSlots, slot)
		}

	case Weekly:
		startTime := time.Date(now.AddDate(0, 0, -6).Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, time.Local)
		for _, view := range views {
			localTime := view.ViewedAt.In(time.Local)
			slot := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), 0, 0, 0, 0, time.Local)
			viewsByTimeSlot[slot]++
		}

		for i := 0; i < 7; i++ {
			slot := startTime.AddDate(0, 0, i)
			timeSlots = append(timeSlots, slot)
		}

	case Monthly:
		currentYear := now.Year()
		currentMonth := now.Month()

		for _, view := range views {
			localTime := view.ViewedAt.In(time.Local)
			slot := time.Date(localTime.Year(), localTime.Month(), 1, 0, 0, 0, 0, time.Local)
			viewsByTimeSlot[slot]++
		}

		for i := 0; i < 12; i++ {
			offsetMonth := int(currentMonth) - 11 + i
			year := currentYear

			for offsetMonth <= 0 {
				offsetMonth += 12
				year--
			}
			for offsetMonth > 12 {
				offsetMonth -= 12
				year++
			}

			slot := time.Date(year, time.Month(offsetMonth), 1, 0, 0, 0, 0, time.Local)
			timeSlots = append(timeSlots, slot)
		}
	}

	var series []analytics.TimeSeriesPoint
	for _, slot := range timeSlots {
		series = append(series, analytics.TimeSeriesPoint{
			Timestamp: slot,
			ViewCount: viewsByTimeSlot[slot],
		})
	}

	return series
}
