package domain

import (
	"encoding/json"
	"errors"
	"fmt"
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
	return nil
}
