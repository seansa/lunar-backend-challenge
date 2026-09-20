package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidMessage = errors.New("invalid message")

const (
	MessageRocketLaunched       = "RocketLaunched"
	MessageRocketSpeedIncreased = "RocketSpeedIncreased"
	MessageRocketSpeedDecreased = "RocketSpeedDecreased"
	MessageRocketExploded       = "RocketExploded"
	MessageRocketMissionChanged = "RocketMissionChanged"
)

type Event struct {
	Channel string
	Number  int64
	Type    string
	Time    time.Time
	Payload json.RawMessage
}

func (e Event) GetKey() string {
	return fmt.Sprintf("%s#%d", e.Channel, e.Number)
}

type Message struct {
	Metadata Metadata
	Message  json.RawMessage
}

type Metadata struct {
	Channel       string
	MessageNumber int64
	MessageTime   time.Time
	MessageType   string
}

func (m Message) Validate() error {
	channel := strings.TrimSpace(m.Metadata.Channel)
	if channel == "" {
		return invalid("metadata.channel is required")
	}
	if m.Metadata.MessageNumber < 1 {
		return invalid("metadata.messageNumber must be >= 1, got %d", m.Metadata.MessageNumber)
	}
	if m.Metadata.MessageTime.IsZero() {
		return invalid("metadata.messageTime is required")
	}
	if m.Message == nil || string(m.Message) == "null" {
		return invalid("message is required")
	}

	return nil
}

func invalid(format string, args ...any) error {
	return &ValidationError{Reason: fmt.Sprintf(format, args...)}
}

type ValidationError struct {
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s", e.Reason)
}
