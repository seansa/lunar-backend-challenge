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

const eventColumns = `channel, message_number, message_type, message_time, payload`
const selectEvents = `SELECT ` + eventColumns + ` FROM events WHERE channel = ? ORDER BY message_number ASC`
const selectEventsAfter = `SELECT ` + eventColumns + ` FROM events WHERE channel = ? AND message_number > ? ORDER BY message_number ASC`

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
	rows, err := r.db.QueryContext(ctx, selectEvents, channel)
	if err != nil {
		return nil, fmt.Errorf("list events for channel %s: %w", channel, err)
	}
	defer rows.Close()

	events, err := scanEvents(rows)
	if err != nil {
		return nil, fmt.Errorf("scan events for channel %s: %w", channel, err)
	}
	return events, nil
}

func (r *Repository) EventsAfter(ctx context.Context, channel string, afterNumber int64) ([]domain.Event, error) {
	rows, err := r.db.QueryContext(ctx, selectEventsAfter, channel, afterNumber)
	if err != nil {
		return nil, fmt.Errorf("list events after %d for channel %s: %w", afterNumber, channel, err)
	}
	defer rows.Close()

	events, err := scanEvents(rows)
	if err != nil {
		return nil, fmt.Errorf("scan events after %d for channel %s: %w", afterNumber, channel, err)
	}
	return events, nil
}

type rowsScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanEvents(rows rowsScanner) ([]domain.Event, error) {
	events := make([]domain.Event, 0)
	for rows.Next() {
		var (
			event   domain.Event
			payload []byte
		)
		if err := rows.Scan(
			&event.Channel,
			&event.Number,
			&event.Type,
			&event.Time,
			&payload,
		); err != nil {
			return nil, err
		}
		event.Time = event.Time.UTC()
		event.Payload = append(event.Payload[:0], payload...)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func keyAlreadyExists(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateKey
}
