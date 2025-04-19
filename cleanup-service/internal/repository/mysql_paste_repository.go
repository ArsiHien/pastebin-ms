package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type MySQLPasteRepository interface {
	Delete(ctx context.Context, url string) error
}

type MySQLPasteRepositoryImpl struct {
	db *sql.DB
}

func NewMySQLPasteRepository(db *sql.DB) MySQLPasteRepository {
	return &MySQLPasteRepositoryImpl{db: db}
}

func (r *MySQLPasteRepositoryImpl) Delete(ctx context.Context, url string) error {
	query := "DELETE FROM pastes WHERE url = ?"
	result, err := r.db.ExecContext(ctx, query, url)
	if err != nil {
		return fmt.Errorf("failed to delete paste %s: %w", url, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("no paste found with url %s", url)
	}
	return nil
}
