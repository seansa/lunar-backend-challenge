package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) HandleEvent(c *gin.Context) {
	var request message
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON or empty request body", "details": err.Error()})
		return
	}

	message := request.toDomainMessage()
	if err := message.Validate(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, request)
}
