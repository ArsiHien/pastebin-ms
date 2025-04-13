package paste

type EventPublisher interface {
	PublishPasteCreated(paste *Paste) error
	Close() error
}
