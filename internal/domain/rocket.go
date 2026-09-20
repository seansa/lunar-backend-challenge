package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type Status string

const (
	// StatusUnknown is used when a channel produced events but no RocketLaunched.
	StatusUnknown  Status = "unknown"
	StatusLaunched Status = "launched"
	StatusExploded Status = "exploded"
)

// Payloads of the known message types.
type (
	LaunchedPayload struct {
		Type        string `json:"type"`
		LaunchSpeed int64  `json:"launchSpeed"`
		Mission     string `json:"mission"`
	}

	SpeedChangedPayload struct {
		By int64 `json:"by"`
	}

	ExplodedPayload struct {
		Reason string `json:"reason"`
	}

	MissionChangedPayload struct {
		NewMission string `json:"newMission"`
	}
)

// Rocket is the aggregate of one channel, rebuilt by folding its events in message-number order.
type Rocket struct {
	Channel           string
	Type              string
	Mission           string
	Speed             int64
	LastMessageNumber int64
	LaunchSpeed       int64
	Status            Status
	ExplosionReason   string
	LaunchedAt        time.Time
	ExplodedAt        time.Time
	MissionChanges    int
	LastMessageTime   time.Time
	EventsApplied     int
	UpdatedAt         time.Time
}

// ApplyEvent folds a single event into the aggregate.
func (r *Rocket) ApplyEvent(e Event) error {
	r.Channel = e.Channel
	if r.Status == "" {
		r.Status = StatusUnknown
	}

	switch e.Type {
	case MessageRocketLaunched:
		var p LaunchedPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return fmt.Errorf("decode %s payload: %w", e.Type, err)
		}
		// A rocket launches once; a re-launch is stored in the log but does not
		// reset already accumulated speed or mission changes. StatusUnknown is
		// the only status a rocket has before its launch.
		if r.Status == StatusUnknown {
			r.Type = p.Type
			r.Mission = p.Mission
			r.LaunchSpeed = p.LaunchSpeed
			r.Speed = p.LaunchSpeed
			r.LaunchedAt = e.Time
			r.Status = StatusLaunched
		}

	case MessageRocketSpeedIncreased:
		var p SpeedChangedPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return fmt.Errorf("decode %s payload: %w", e.Type, err)
		}
		r.Speed += p.By

	case MessageRocketSpeedDecreased:
		var p SpeedChangedPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return fmt.Errorf("decode %s payload: %w", e.Type, err)
		}
		r.Speed -= p.By

	case MessageRocketExploded:
		var p ExplodedPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return fmt.Errorf("decode %s payload: %w", e.Type, err)
		}
		if r.Status != StatusExploded {
			r.Status = StatusExploded
			r.ExplosionReason = p.Reason
			r.ExplodedAt = e.Time
		}

	case MessageRocketMissionChanged:
		var p MissionChangedPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return fmt.Errorf("decode %s payload: %w", e.Type, err)
		}
		r.Mission = p.NewMission
		r.MissionChanges++

	default:
		return fmt.Errorf("unknown message type %q", e.Type)
	}

	r.LastMessageNumber = e.Number
	r.LastMessageTime = e.Time
	r.EventsApplied++
	return nil
}
