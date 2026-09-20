package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/seansa/lunar-backend-challenge/internal/consumer"
	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/service"
	"github.com/seansa/lunar-backend-challenge/internal/store/rocket"
)

type Processor interface {
	Process(ctx context.Context, message domain.Message) (service.Result, error)
}

type Dispatcher interface {
	Dispatch(ctx context.Context, msg domain.Message) (consumer.Result, error)
}

type RocketReader interface {
	Rockets(ctx context.Context, opts rocket.ListOptions) ([]domain.Rocket, error)
	Rocket(ctx context.Context, channel string) (domain.Rocket, error)
}

type EventReader interface {
	Events(ctx context.Context, channel string) ([]domain.Event, error)
}

type Handler struct {
	service      Processor
	dispatcher   Dispatcher
	rocketReader RocketReader
	eventReader  EventReader
}

func New(service Processor, dispatcher Dispatcher, rocketReader RocketReader, eventReader EventReader) *Handler {
	return &Handler{
		service:      service,
		dispatcher:   dispatcher,
		rocketReader: rocketReader,
		eventReader:  eventReader,
	}
}

func (h *Handler) Process(c *gin.Context) {
	var request message
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON or empty request body", "details": err.Error()})
		return
	}

	message := request.toDomainMessage()

	if err := message.Validate(); err != nil {
		respondValidation(c, err)
		return
	}

	result, err := h.dispatcher.Dispatch(c, message)
	if err != nil {
		respondConsumerError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, result)
}

func (h *Handler) HandleEvent(c *gin.Context) {
	var request message
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON or empty request body", "details": err.Error()})
		return
	}

	message := request.toDomainMessage()
	if err := message.Validate(); err != nil {
		respondValidation(c, err)
		return
	}

	result, err := h.service.Process(c, message)
	switch {
	case errors.Is(err, domain.ErrInvalidMessage):
		c.JSON(http.StatusBadRequest, fmt.Sprintf("message is not valid	%s", err.Error()))
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to consume message", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, acceptedResponse{
		Status:        "accepted",
		Channel:       result.Channel,
		MessageNumber: result.MessageNumber,
	})
}

func (h *Handler) ListRockets(c *gin.Context) {
	sortField, ok := rocket.ParseSortField(c.DefaultQuery("sort", string(rocket.SortByChannel)))
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid_parameter", "sort must be one of id, type, mission, status")
		return
	}

	descending := false
	switch strings.ToLower(c.DefaultQuery("order", "asc")) {
	case "asc":
	case "desc":
		descending = true
	default:
		respondError(c, http.StatusBadRequest, "invalid_parameter", "order must be asc or desc")
		return
	}

	rockets, err := h.rocketReader.Rockets(c, rocket.ListOptions{
		Sort:       sortField,
		Descending: descending,
	})
	if err != nil {
		slog.Error("list rockets failed", "error", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "rockets could not be listed")
		return
	}

	response := rocketListResponse{Count: len(rockets), Rockets: make([]rocketResponse, 0, len(rockets))}
	for _, rocket := range rockets {
		response.Rockets = append(response.Rockets, toRocketResponse(rocket))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetRocket(c *gin.Context) {
	channel := c.Param("channel")

	response, err := h.rocketReader.Rocket(c, channel)
	switch {
	case errors.Is(err, rocket.ErrRocketNotFound):
		respondError(c, http.StatusNotFound, "rocket_not_found", "no rocket with that channel")
		return
	case err != nil:
		slog.Error("get rocket failed", "channel", channel, "error", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "the rocket could not be read")
		return
	}

	c.JSON(http.StatusOK, toRocketResponse(response))
}

func (h *Handler) ListEvents(c *gin.Context) {
	channel := c.Param("channel")

	events, err := h.eventReader.Events(c, channel)
	if err != nil {
		slog.Error("list events failed", "channel", channel, "error", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "the events could not be read")
		return
	}
	if len(events) == 0 {
		respondError(c, http.StatusNotFound, "rocket_not_found", "no rocket with that channel")
		return
	}

	response := eventListResponse{Channel: channel, Count: len(events), Events: make([]eventResponse, 0, len(events))}
	for _, event := range events {
		response.Events = append(response.Events, toEventResponse(event))
	}

	c.JSON(http.StatusOK, response)
}
