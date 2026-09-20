package handler

import (
	"encoding/json"
	"time"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
)

type message struct {
	Metadata metadata        `json:"metadata"`
	Message  json.RawMessage `json:"message"`
}

type metadata struct {
	Channel       string    `json:"channel" binding:"required"`
	MessageNumber int64     `json:"messageNumber" binding:"required"`
	MessageTime   time.Time `json:"messageTime" binding:"required"`
	MessageType   string    `json:"messageType" binding:"required"`
}

func (im message) toDomainMessage() domain.Message {
	return domain.Message{
		Metadata: domain.Metadata{
			Channel:       im.Metadata.Channel,
			MessageNumber: im.Metadata.MessageNumber,
			MessageTime:   im.Metadata.MessageTime,
			MessageType:   im.Metadata.MessageType,
		},
		Message: im.Message,
	}
}
