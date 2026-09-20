package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestApplyEventFoldsTheStream(t *testing.T) {
	rocket := fold(
		event(1, MessageRocketLaunched, `{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}`),
		event(2, MessageRocketSpeedIncreased, `{"by":3000}`),
		event(3, MessageRocketSpeedDecreased, `{"by":2500}`),
		event(4, MessageRocketMissionChanged, `{"newMission":"SHUTTLE_MIR"}`),
		event(5, MessageRocketExploded, `{"reason":"PRESSURE_VESSEL_FAILURE"}`),
	)

	assertEqual(t, "type", rocket.Type, "Falcon-9")
	assertEqual(t, "mission", rocket.Mission, "SHUTTLE_MIR")
	assertEqual(t, "speed", rocket.Speed, int64(1000))
	assertEqual(t, "mission changes", rocket.MissionChanges, 1)
	assertEqual(t, "status", rocket.Status, StatusExploded)
	assertEqual(t, "explosion reason", rocket.ExplosionReason, "PRESSURE_VESSEL_FAILURE")
	assertEqual(t, "last message number", rocket.LastMessageNumber, int64(5))
	assertEqual(t, "events applied", rocket.EventsApplied, 5)
}

func TestApplyEventSkipsMessagesItCannotFold(t *testing.T) {
	var rocket Rocket

	assertEqual(t, "launch folded", rocket.ApplyEvent(event(1, MessageRocketLaunched, `{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}`)), true)
	assertEqual(t, "unknown type folded", rocket.ApplyEvent(event(2, "RocketDocked", `{"with":"FOO"}`)), false)
	assertEqual(t, "broken payload folded", rocket.ApplyEvent(event(3, MessageRocketSpeedIncreased, `{"by":"a lot"}`)), false)

	// The unknown type and the broken payload are consumed instead of folded, so
	// the stream keeps moving and one bad message cannot block the channel.
	assertEqual(t, "speed", rocket.Speed, int64(500))
	assertEqual(t, "status", rocket.Status, StatusLaunched)
	assertEqual(t, "last message number", rocket.LastMessageNumber, int64(3))
	assertEqual(t, "events applied", rocket.EventsApplied, 3)
}

func fold(events ...Event) Rocket {
	var rocket Rocket
	for _, e := range events {
		rocket.ApplyEvent(e)
	}
	return rocket
}

func event(number int64, messageType, payload string) Event {
	return Event{
		Channel: "channel-1",
		Number:  number,
		Type:    messageType,
		Time:    time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC).Add(time.Duration(number) * time.Second),
		Payload: json.RawMessage(payload),
	}
}

func assertEqual[T comparable](t *testing.T, field string, got, want T) {
	t.Helper()

	if got != want {
		t.Fatalf("%s = %v, want %v", field, got, want)
	}
}
