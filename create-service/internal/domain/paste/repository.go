package paste

type Repository interface {
	Save(paste *Paste) error
}
