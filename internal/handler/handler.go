package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/seansa/lunar-backend-challenge/internal/domain"
	"github.com/seansa/lunar-backend-challenge/internal/service"
)

type Processor interface {
	Process(ctx context.Context, message domain.Message) (service.Result, error)
}

type Handler struct {
	service Processor
}

func New(service Processor) *Handler {
	return &Handler{
		service: service,
	}
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
