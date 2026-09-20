package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/seansa/lunar-backend-challenge/internal/consumer"
	"github.com/seansa/lunar-backend-challenge/internal/domain"
)

type errorBody struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []errorDetail `json:"details,omitempty"`
}

type errorDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func respondError(c *gin.Context, status int, code, message string, details ...errorDetail) {
	c.AbortWithStatusJSON(status, errorBody{Error: apiError{Code: code, Message: message, Details: details}})
}

func respondValidation(c *gin.Context, err error) {
	if invalid, ok := errors.AsType[*domain.ValidationError](err); ok {
		respondError(c, http.StatusUnprocessableEntity, "invalid_message", "the message was rejected",
			errorDetail{Reason: invalid.Reason})
		return
	}
	respondError(c, http.StatusBadRequest, "invalid_message", err.Error())
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
