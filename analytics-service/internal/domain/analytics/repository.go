package analytics

import "context"

type Repository interface {
	SaveView(ctx context.Context, view *View) error
	IncrementViewCount(ctx context.Context, pasteURL string) error
	GetAnalytics(ctx context.Context, pasteURL string,
		period string) ([]View, error)
	GetPastesStats(ctx context.Context) (map[string]int, error)
	GetViewCount(ctx context.Context, pasteURL string) (int, error)
}
