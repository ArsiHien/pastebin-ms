package repository

import (
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"gorm.io/gorm"
)

type PasteMySQLRepository struct {
	db *gorm.DB
}

func NewPasteMySQLRepository(db *gorm.DB) *PasteMySQLRepository {
	return &PasteMySQLRepository{db: db}
}

func (r *PasteMySQLRepository) Save(paste *paste.Paste) error {
	return r.db.Create(paste).Error
}
