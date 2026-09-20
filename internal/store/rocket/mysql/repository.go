package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
)

const rocketInsertColumns = `
channel, rocket_type, mission, mission_changes, speed, launch_speed, status, exploded_reason,
last_message_number, last_message_time, launched_at, exploded_at, events_applied`

const rocketSelectColumns = rocketInsertColumns + `, updated_at`

const selectRocket = `SELECT` + rocketSelectColumns + ` FROM rockets WHERE channel = ?`

const upsertRocket = `
INSERT INTO rockets (` + rocketInsertColumns + `)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	rocket_type = ?, mission = ?, mission_changes = ?, speed = ?, launch_speed = ?, status = ?,
	exploded_reason = ?, last_message_number = ?, last_message_time = ?, launched_at = ?, exploded_at = ?,
	events_applied = ?`

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, channel string) (domain.Rocket, bool, error) {
	row := r.db.QueryRowContext(ctx, selectRocket, channel)
	rocket, err := scanRocket(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Rocket{}, false, nil
	}
	if err != nil {
		return domain.Rocket{}, false, fmt.Errorf("get rocket %s: %w", channel, err)
	}
	return rocket, true, nil
}

func (r *Repository) Upsert(ctx context.Context, rocket domain.Rocket) error {
	args := rocketArgs(rocket)
	args = append(args, args[1:]...) // the update clause binds the same values

	if _, err := r.db.ExecContext(ctx, upsertRocket, args...); err != nil {
		return fmt.Errorf("upsert rocket %s: %w", rocket.Channel, err)
	}
	return nil
}

func rocketArgs(rocket domain.Rocket) []any {
	return []any{
		rocket.Channel,
		rocket.Type,
		rocket.Mission,
		rocket.MissionChanges,
		rocket.Speed,
		rocket.LaunchSpeed,
		string(rocket.Status),
		rocket.ExplosionReason,
		rocket.LastMessageNumber,
		utcOrNullTime(rocket.LastMessageTime),
		utcOrNullTime(rocket.LaunchedAt),
		utcOrNullTime(rocket.ExplodedAt),
		rocket.EventsApplied,
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRocket(row rowScanner) (domain.Rocket, error) {
	var (
		rocket          domain.Rocket
		status          string
		lastMessageTime sql.NullTime
		launchedAt      sql.NullTime
		explodedAt      sql.NullTime
		updatedAt       sql.NullTime
	)

	err := row.Scan(
		&rocket.Channel,
		&rocket.Type,
		&rocket.Mission,
		&rocket.MissionChanges,
		&rocket.Speed,
		&rocket.LaunchSpeed,
		&status,
		&rocket.ExplosionReason,
		&rocket.LastMessageNumber,
		&lastMessageTime,
		&launchedAt,
		&explodedAt,
		&rocket.EventsApplied,
		&updatedAt,
	)
	if err != nil {
		return domain.Rocket{}, err
	}

	rocket.Status = domain.Status(status)
	rocket.LastMessageTime = lastMessageTime.Time.UTC()
	rocket.LaunchedAt = launchedAt.Time.UTC()
	rocket.ExplodedAt = explodedAt.Time.UTC()
	rocket.UpdatedAt = updatedAt.Time.UTC()
	return rocket, nil
}

func utcOrNullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
