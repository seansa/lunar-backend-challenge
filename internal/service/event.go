package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/store/event"
	"github.com/seansa/lunar-backend-challenge/pkg/lock"
)

// Result describes what happened to an inbound message. It is serialised by the
// synchronous endpoint, so it mirrors consumer.Result field by field.
type Result struct {
	Channel       string `json:"channel"`
	MessageNumber int64  `json:"messageNumber"`
	MessageType   string `json:"messageType"`
	Duplicate     bool   `json:"duplicate"`
	Applied       bool   `json:"applied"`
}

//go:generate go tool mockgen -source=event.go -destination=../mocks/event_store.go -package=mocks -typed -mock_names=eventStore=MockEventStore

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
	incomingEvent, err := domain.NewEvent(message)
	if err != nil {
		return Result{}, err
	}

	duplicate := false
	switch err := s.store.Append(ctx, incomingEvent); {
	case errors.Is(err, event.ErrDuplicate):
		duplicate = true
	case err != nil:
		return Result{}, fmt.Errorf("append event: %w", err)
	}

	applied, err := s.ApplyEvent(ctx, incomingEvent)
	if err != nil {
		return Result{}, fmt.Errorf("apply event to rocket state: %w", err)
	}

	return Result{
		Channel:       incomingEvent.Channel,
		MessageNumber: incomingEvent.Number,
		MessageType:   string(incomingEvent.Type),
		Duplicate:     duplicate,
		Applied:       applied,
	}, nil
}

func (s *Service) Events(ctx context.Context, channel string) ([]domain.Event, error) {
	return s.store.Events(ctx, channel)
}
