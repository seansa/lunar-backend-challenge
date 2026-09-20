package consumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
	eventstore "github.com/seansa/lunar-backend-challenge/internal/store/event"
)

// Result describes what happened to an inbound message.
type Result struct {
	Channel       string    `json:"channel"`
	MessageNumber int64     `json:"messageNumber"`
	MessageType   string    `json:"messageType"`
	Duplicate     bool      `json:"duplicate"`
	Applied       bool      `json:"applied"`
	ReceivedAt    time.Time `json:"receivedAt"`
}

// Applier folds a stored event into the rocket state.
type Applier interface {
	ApplyEvent(ctx context.Context, event domain.Event) (bool, error)
}

type store interface {
	Append(ctx context.Context, e domain.Event) error
}

// Service turns an inbound message into a stored event and updates the rocket state.
type Service struct {
	store store
	state Applier
	now   func() time.Time
}

func New(store store, state Applier) *Service {
	return &Service{store: store, state: state, now: time.Now}
}

// Process appends the message to the event log and updates the rocket state.
//
// At-least-once delivery means the same message can be processed twice. A
// duplicate append is not an error: folding an event is idempotent, so replaying
// it either applies the event (when a previous attempt failed after the append)
// or does nothing.
func (s *Service) Process(ctx context.Context, msg domain.Message) (Result, error) {

	event := domain.Event{
		Channel: msg.Metadata.Channel,
		Number:  msg.Metadata.MessageNumber,
		Type:    msg.Metadata.MessageType,
		Time:    msg.Metadata.MessageTime,
		Payload: msg.Message,
	}

	duplicate := false
	switch err := s.store.Append(ctx, event); {
	case errors.Is(err, eventstore.ErrDuplicate):
		duplicate = true
	case err != nil:
		return Result{}, fmt.Errorf("append event: %w", err)
	}

	applied, err := s.state.ApplyEvent(ctx, event)
	if err != nil {
		return Result{}, fmt.Errorf("apply event to rocket state: %w", err)
	}

	return Result{
		Channel:       event.Channel,
		MessageNumber: event.Number,
		MessageType:   event.Type,
		Duplicate:     duplicate,
		Applied:       applied,
	}, nil
}
