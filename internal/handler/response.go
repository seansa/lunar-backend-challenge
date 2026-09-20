package handler

import (
	"encoding/json"
	"time"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
)

type acceptedResponse struct {
	Status        string `json:"status"`
	Channel       string `json:"channel"`
	MessageNumber int64  `json:"messageNumber"`
}

type rocketResponse struct {
	Channel           string     `json:"channel"`
	Type              string     `json:"type"`
	Mission           string     `json:"mission"`
	MissionChanges    int        `json:"missionChanges"`
	Speed             int64      `json:"speed"`
	LaunchSpeed       int64      `json:"launchSpeed"`
	Status            string     `json:"status"`
	ExplodedReason    *string    `json:"explodedReason"`
	LastMessageNumber int64      `json:"lastMessageNumber"`
	LastMessageTime   *time.Time `json:"lastMessageTime"`
	LaunchedAt        *time.Time `json:"launchedAt"`
	ExplodedAt        *time.Time `json:"explodedAt"`
	EventsApplied     int        `json:"eventsApplied"`
	UpdatedAt         *time.Time `json:"updatedAt"`
}

type rocketListResponse struct {
	Count   int              `json:"count"`
	Rockets []rocketResponse `json:"rockets"`
}

type eventListResponse struct {
	Channel string          `json:"channel"`
	Count   int             `json:"count"`
	Events  []eventResponse `json:"events"`
}

type eventResponse struct {
	MessageNumber int             `json:"messageNumber"`
	MessageType   string          `json:"messageType"`
	MessageTime   time.Time       `json:"messageTime"`
	Payload       json.RawMessage `json:"payload"`
}

func toEventResponse(event domain.Event) eventResponse {
	return eventResponse{
		MessageNumber: int(event.Number),
		MessageType:   event.Type,
		MessageTime:   event.Time.UTC(),
		Payload:       event.Payload,
	}
}

func toRocketResponse(rocket domain.Rocket) rocketResponse {
	return rocketResponse{
		Channel:           rocket.Channel,
		Type:              rocket.Type,
		Mission:           rocket.Mission,
		MissionChanges:    rocket.MissionChanges,
		Speed:             rocket.Speed,
		LaunchSpeed:       rocket.LaunchSpeed,
		Status:            string(rocket.Status),
		ExplodedReason:    optionalString(rocket.ExplosionReason),
		LastMessageNumber: rocket.LastMessageNumber,
		LastMessageTime:   optionalTime(rocket.LastMessageTime),
		LaunchedAt:        optionalTime(rocket.LaunchedAt),
		ExplodedAt:        optionalTime(rocket.ExplodedAt),
		EventsApplied:     rocket.EventsApplied,
		UpdatedAt:         optionalTime(rocket.UpdatedAt),
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}
