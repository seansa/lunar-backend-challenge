package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/seansa/lunar-backend-challenge/internal/consumer"
	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/service"
)

type Processor interface {
	Process(ctx context.Context, message domain.Message) (service.Result, error)
}

type Dispatcher interface {
	Dispatch(ctx context.Context, msg domain.Message) (consumer.Result, error)
}

type Handler struct {
	service    Processor
	dispatcher Dispatcher
}

func New(service Processor, dispatcher Dispatcher) *Handler {
	return &Handler{
		service:    service,
		dispatcher: dispatcher,
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

// respondConsumerError maps consumer failures to status codes the sender can act on.
// Non-2xx answers make the rocket redeliver, which is what we want for the
// transient cases: at-least-once delivery plus deduplication make retries safe.
func respondConsumerError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, consumer.ErrPoolClosed):
		c.Header("Retry-After", "1")
		respondError(c, http.StatusServiceUnavailable, "service_shutting_down",
			"the service is shutting down, retry the message")
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		c.Header("Retry-After", "1")
		respondError(c, http.StatusServiceUnavailable, "consumer_timeout", "the message could not be consumed in time")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "the message could not be stored")
	}
}
