package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/store/event"
)

const insertEvent = `INSERT INTO events (channel, message_number, message_type, message_time, payload) VALUES (?, ?, ?, ?, ?)`

const mysqlDuplicateKey = 1062

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Append(ctx context.Context, e domain.Event) error {
	_, err := r.db.ExecContext(ctx, insertEvent,
		e.Channel, e.Number, e.Type, e.Time.UTC(), string(e.Payload))
	if err == nil {
		return nil
	}
	if keyAlreadyExists(err) {
		return event.ErrDuplicate
	}
	return fmt.Errorf("insert event: %w", err)
}

func (r *Repository) Events(ctx context.Context, channel string) ([]domain.Event, error) {
	return nil, nil
}

func (r *Repository) EventsAfter(ctx context.Context, channel string, afterNumber int64) ([]domain.Event, error) {
	return nil, nil
}

func keyAlreadyExists(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateKey
}
