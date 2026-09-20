package domain

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
}

// ApplyEvent folds a single event into the aggregate.
func (r *Rocket) ApplyEvent(e Event) error {
	return nil
}
