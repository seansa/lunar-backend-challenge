package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/mocks"
	"go.uber.org/mock/gomock"
)

const testChannel = "channel-1"

func TestApplyEventFoldsTheMessageAndDrainsTheLog(t *testing.T) {
	var saved domain.Rocket

	ctrl := gomock.NewController(t)
	store := mocks.NewMockEventStore(ctrl)
	projection := mocks.NewMockRocketProjection(ctrl)

	ctx := context.Background()
	launched := domain.Rocket{
		Channel: testChannel, Type: "Falcon-9", Mission: "ARTEMIS", Speed: 500,
		LaunchSpeed: 500, Status: domain.StatusLaunched, LastMessageNumber: 1, EventsApplied: 1,
	}

	projection.EXPECT().Get(ctx, testChannel).Return(launched, true, nil)
	// Message 3 already arrived and is waiting in the log behind message 2.
	store.EXPECT().EventsAfter(ctx, testChannel, int64(2)).Return([]domain.Event{speedChanged(3, 3000)}, nil)
	projection.EXPECT().Upsert(ctx, gomock.Any()).Do(func(_ context.Context, rocket domain.Rocket) error {
		saved = rocket
		return nil
	})

	applied, err := New(store, projection).ApplyEvent(ctx, speedChanged(2, 1000))
	if err != nil {
		t.Fatalf("ApplyEvent() error = %v", err)
	}

	if !applied {
		t.Fatal("the message must be applied")
	}
	if want := int64(4500); saved.Speed != want {
		t.Fatalf("speed = %d, want %d", saved.Speed, want)
	}
	if want := int64(3); saved.LastMessageNumber != want {
		t.Fatalf("last message number = %d, want %d", saved.LastMessageNumber, want)
	}
}

func TestApplyEventWaitsWhileTheStreamStartsWithAHole(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockEventStore(ctrl)
	projection := mocks.NewMockRocketProjection(ctrl)

	ctx := context.Background()

	projection.EXPECT().Get(ctx, testChannel).Return(domain.Rocket{}, false, nil)
	// The launch is still missing, so there is no state to project yet.
	store.EXPECT().Events(ctx, testChannel).Return([]domain.Event{speedChanged(3, 3000)}, nil)

	applied, err := New(store, projection).ApplyEvent(ctx, speedChanged(3, 3000))
	if err != nil {
		t.Fatalf("ApplyEvent() error = %v", err)
	}
	if applied {
		t.Fatal("nothing can be projected while the stream starts with a hole")
	}
}

func TestApplyEventIgnoresRedeliveries(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockEventStore(ctrl)
	projection := mocks.NewMockRocketProjection(ctrl)

	ctx := context.Background()
	current := domain.Rocket{Channel: testChannel, Status: domain.StatusLaunched, LastMessageNumber: 5}

	projection.EXPECT().Get(ctx, testChannel).Return(current, true, nil)

	applied, err := New(store, projection).ApplyEvent(ctx, speedChanged(3, 3000))
	if err != nil {
		t.Fatalf("ApplyEvent() error = %v", err)
	}
	if applied {
		t.Fatal("a message that was already folded must not be applied again")
	}
}

func speedChanged(number, by int64) domain.Event {
	return domain.Event{
		Channel: testChannel,
		Number:  number,
		Type:    domain.MessageRocketSpeedIncreased,
		Payload: json.RawMessage(fmt.Sprintf(`{"by":%d}`, by)),
	}
}
