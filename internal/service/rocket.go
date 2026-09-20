package service

import (
	"context"
	"log/slog"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/store/rocket"
)

//go:generate go tool mockgen -source=rocket.go -destination=../mocks/rocket_projection.go -package=mocks -typed -mock_names=rocketProjection=MockRocketProjection

type rocketProjection interface {
	Get(ctx context.Context, channel string) (domain.Rocket, bool, error)
	Upsert(ctx context.Context, r domain.Rocket) error
	List(ctx context.Context, opts rocket.ListOptions) ([]domain.Rocket, error)
}

const (
	appliedEvent    = true
	notAppliedEvent = false
)

func (s *Service) ApplyEvent(ctx context.Context, e domain.Event) (bool, error) {
	// Folding one channel is serialised: the read-modify-write below has to stay
	// inside the lock, otherwise a slow worker can overwrite a newer state.
	release := s.locks.Lock(e.Channel)
	defer release()

	current, found, err := s.projection.Get(ctx, e.Channel)
	if err != nil {
		return notAppliedEvent, err
	}
	if found && e.Number <= current.LastMessageNumber {
		// Already applied
		return notAppliedEvent, nil
	}

	rocket := current
	if !found {
		rocket = domain.Rocket{Channel: e.Channel}
	}

	if e.Number == rocket.LastMessageNumber+1 {
		if !rocket.ApplyEvent(e) {
			logSkipped(e)
		}

		pending, err := s.store.EventsAfter(ctx, e.Channel, rocket.LastMessageNumber)
		if err != nil {
			return notAppliedEvent, err
		}

		applyInOrder(&rocket, pending)

		if err := s.projection.Upsert(ctx, rocket); err != nil {
			return notAppliedEvent, err
		}
		return appliedEvent, nil
	}

	rebuilt, hasState, err := s.rebuildFromStore(ctx, e.Channel)
	if err != nil {
		return notAppliedEvent, err
	}
	if !hasState {
		// The stream still starts with a hole, so there is no state to store yet.
		return notAppliedEvent, nil
	}
	if err := s.projection.Upsert(ctx, rebuilt); err != nil {
		return notAppliedEvent, err
	}

	// The rebuild rewrites the state even when the message we received is not the
	// next one, so report whether the projection actually moved forward.
	return rebuilt.LastMessageNumber > current.LastMessageNumber, nil
}

func (s Service) Rockets(ctx context.Context, opts rocket.ListOptions) ([]domain.Rocket, error) {
	return s.projection.List(ctx, opts)
}

func (s Service) Rocket(ctx context.Context, channel string) (domain.Rocket, error) {
	result, found, err := s.projection.Get(ctx, channel)
	if err != nil {
		return domain.Rocket{}, err
	}
	if !found {
		return domain.Rocket{}, rocket.ErrRocketNotFound
	}
	return result, nil
}

func (s *Service) rebuildFromStore(ctx context.Context, channel string) (domain.Rocket, bool, error) {
	events, err := s.store.Events(ctx, channel)
	if err != nil {
		return domain.Rocket{}, false, err
	}

	rocket := domain.Rocket{Channel: channel}
	applyInOrder(&rocket, events)

	return rocket, rocket.LastMessageNumber > 0, nil
}

// applyInOrder folds every event that extends the contiguous run of the stream,
// so messages that arrived early are folded as soon as their predecessors are
// there.
func applyInOrder(rocket *domain.Rocket, events []domain.Event) {
	for _, e := range events {
		if e.Number <= rocket.LastMessageNumber {
			continue
		}
		if e.Number != rocket.LastMessageNumber+1 {
			break
		}
		if !rocket.ApplyEvent(e) {
			logSkipped(e)
		}
	}
}

// logSkipped records a message we could not fold. It is not an error: the event
// is stored, and the projection moves on.
func logSkipped(e domain.Event) {
	slog.Warn("event skipped", "channel", e.Channel, "message_number", e.Number, "message_type", e.Type)
}
