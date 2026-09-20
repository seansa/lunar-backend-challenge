package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
)

type fakeEventRows struct {
	events []domain.Event
	index  int
	err    error
}

func (r *fakeEventRows) Next() bool {
	return r.index < len(r.events)
}

func (r *fakeEventRows) Scan(dest ...any) error {
	event := r.events[r.index]
	r.index++
	*dest[0].(*string) = event.Channel
	*dest[1].(*int64) = event.Number
	*dest[2].(*string) = event.Type
	*dest[3].(*time.Time) = event.Time
	*dest[4].(*[]byte) = event.Payload
	return nil
}

func (r *fakeEventRows) Err() error {
	return r.err
}

func TestScanEvents(t *testing.T) {
	originalTime := time.Date(2026, 9, 20, 12, 0, 0, 123456000, time.FixedZone("CEST", 2*60*60))
	rows := &fakeEventRows{
		events: []domain.Event{{
			Channel: "rocket-1",
			Number:  2,
			Type:    domain.MessageRocketSpeedIncreased,
			Time:    originalTime,
			Payload: []byte(`{"by":3000}`),
		}},
	}

	events, err := scanEvents(rows)
	if err != nil {
		t.Fatalf("scanEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("scanEvents() returned %d events, want 1", len(events))
	}
	if !events[0].Time.Equal(originalTime.UTC()) {
		t.Fatalf("event time = %s, want %s", events[0].Time, originalTime.UTC())
	}
	if string(events[0].Payload) != `{"by":3000}` {
		t.Fatalf("event payload = %s, want JSON payload", events[0].Payload)
	}
}

func TestScanEventsReturnsRowsError(t *testing.T) {
	wantErr := errors.New("rows failed")

	_, err := scanEvents(&fakeEventRows{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("scanEvents() error = %v, want %v", err, wantErr)
	}
}
