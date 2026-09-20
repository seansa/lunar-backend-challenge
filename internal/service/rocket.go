package service

import (
	"context"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
)

type rocketProjection interface {
	Get(ctx context.Context, channel string) (domain.Rocket, bool, error)
	Upsert(ctx context.Context, r domain.Rocket) error
}

const (
	appliedEvent    = true
	notAppliedEvent = false
)

func (s *Service) ApplyEvent(ctx context.Context, e domain.Event) (bool, error) {
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
		if err := rocket.ApplyEvent(e); err != nil {
			return notAppliedEvent, err
		}

		pending, err := s.store.EventsAfter(ctx, e.Channel, rocket.LastMessageNumber)
		if err != nil {
			return notAppliedEvent, err
		}

		for _, e := range pending {
			if e.Number <= rocket.LastMessageNumber {
				continue
			}
			if e.Number != rocket.LastMessageNumber+1 {
				break
			}
			if err := rocket.ApplyEvent(e); err != nil {
				return notAppliedEvent, err
			}
		}

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
	return appliedEvent, nil
}

func (s *Service) rebuildFromStore(ctx context.Context, channel string) (domain.Rocket, bool, error) {
	events, err := s.store.Events(ctx, channel)
	if err != nil {
		return domain.Rocket{}, false, err
	}

	rocket := domain.Rocket{Channel: channel}

	applied := 0
	for _, e := range events {
		if e.Number <= rocket.LastMessageNumber {
			continue
		}
		if e.Number != rocket.LastMessageNumber+1 {
			break
		}
		if err := rocket.ApplyEvent(e); err != nil {
			return domain.Rocket{}, false, err
		}
		applied++
	}
	return rocket, applied > 0, nil
}
