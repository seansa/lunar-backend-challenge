package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/store/event"
	"github.com/seansa/lunar-backend-challenge/pkg/lock"
)

type Result struct {
	Channel       string
	MessageNumber int64
	MessageType   string
	Applied       bool
	Duplicate     bool
}

type eventStore interface {
	Append(ctx context.Context, e domain.Event) error
	EventsAfter(ctx context.Context, channel string, afterNumber int64) ([]domain.Event, error)
	Events(ctx context.Context, channel string) ([]domain.Event, error)
}

type Service struct {
	store      eventStore
	projection rocketProjection
	locks      *lock.KeyedMutex
}

func New(store eventStore, projection rocketProjection) *Service {
	return &Service{
		store:      store,
		projection: projection,
		locks:      lock.NewKeyedMutex(),
	}
}

func (s *Service) Process(ctx context.Context, message domain.Message) (Result, error) {
	incommingEvent, err := domain.NewEvent(message)
	if err != nil {
		return Result{}, err
	}

	duplicate := false
	switch err := s.store.Append(ctx, incommingEvent); {
	case errors.Is(err, event.ErrDuplicate):
		duplicate = true
	case errors.Is(err, event.ErrConflict):
		return Result{}, event.ErrConflict
	case err != nil:
		return Result{}, fmt.Errorf("append event: %w", err)
	}

	applied, err := s.ApplyEvent(ctx, incommingEvent)
	if err != nil {
		return Result{}, fmt.Errorf("apply event to rocket state: %w", err)
	}

	return Result{
		Channel:       incommingEvent.Channel,
		MessageNumber: incommingEvent.Number,
		MessageType:   string(incommingEvent.Type),
		Applied:       applied,
		Duplicate:     duplicate,
	}, nil
}

func (s *Service) Events(ctx context.Context, channel string) ([]domain.Event, error) {
	return s.store.Events(ctx, channel)
}
