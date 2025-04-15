package paste

type Repository interface {
	FindByURL(url string) (*Paste, error)
	MarkAsRead(url string) error
}
