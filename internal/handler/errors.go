package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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
