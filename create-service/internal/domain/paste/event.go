package paste

import "context"

type EventPublisher interface {
	PublishPasteCreated(paste *Paste) error
	PublishPasteSave(ctx context.Context, pasteData []byte) error
	Close() error
}
