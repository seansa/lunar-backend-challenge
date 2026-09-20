package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/mocks"
	eventstore "github.com/seansa/lunar-backend-challenge/internal/store/event"
	"go.uber.org/mock/gomock"
)

func TestProcessStoresAndFoldsTheMessage(t *testing.T) {
	var stored domain.Event

	ctrl := gomock.NewController(t)
	store := mocks.NewMockStore(ctrl)
	state := mocks.NewMockApplier(ctrl)

	ctx := context.Background()
	store.EXPECT().Append(ctx, gomock.Any()).Do(func(_ context.Context, event domain.Event) error {
		stored = event
		return nil
	})
	state.EXPECT().ApplyEvent(ctx, gomock.Any()).Return(true, nil)

	result, err := New(store, state).Process(ctx, message())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.Channel != "channel-1" || stored.Channel != "channel-1" {
		t.Fatalf("channel = %q, want the trimmed value", stored.Channel)
	}
	if !result.Applied || result.Duplicate {
		t.Fatalf("applied = %t, duplicate = %t, want true, false", result.Applied, result.Duplicate)
	}
	if want := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC); !stored.Time.Equal(want) {
		t.Fatalf("message time = %s, want %s", stored.Time, want)
	}
}

func TestProcessToleratesARedelivery(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStore(ctrl)
	state := mocks.NewMockApplier(ctrl)

	ctx := context.Background()
	store.EXPECT().Append(ctx, gomock.Any()).Return(eventstore.ErrDuplicate)
	state.EXPECT().ApplyEvent(ctx, gomock.Any()).Return(false, nil)

	result, err := New(store, state).Process(ctx, message())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if !result.Duplicate {
		t.Fatal("the redelivery must be reported as a duplicate, not as a failure")
	}
}

func TestProcessRejectsAnInvalidMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStore(ctrl)
	state := mocks.NewMockApplier(ctrl)

	invalid := message()
	invalid.Metadata.Channel = ""

	if _, err := New(store, state).Process(context.Background(), invalid); err == nil {
		t.Fatal("Process() error = nil, want a validation error")
	}
}

func message() domain.Message {
	return domain.Message{
		Metadata: domain.Metadata{
			Channel:       " channel-1 ",
			MessageNumber: 1,
			MessageTime:   time.Date(2026, 9, 20, 12, 0, 0, 0, time.FixedZone("CEST", 2*60*60)),
			MessageType:   domain.MessageRocketLaunched,
		},
		Message: json.RawMessage(`{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}`),
	}
}
