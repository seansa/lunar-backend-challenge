package main

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
)

func newRouter(deps *dependencies) *gin.Engine {
	router := gin.New()

	router.POST("/messages", func(ctx *gin.Context) {
		var rawMessage json.RawMessage
		ctx.ShouldBindJSON(&rawMessage)
		slog.Info(fmt.Sprintf("%v", rawMessage))
	})

	return router
}
