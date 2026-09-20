package main

import (
	"github.com/gin-gonic/gin"
	"github.com/seansa/lunar-backend-challenge/internal/handler"
)

func newRouter(deps *dependencies) *gin.Engine {
	router := gin.New()

	handler := handler.New(deps.service, deps.consumer)

	router.POST("/messages", handler.Process)
	router.POST("/events", handler.HandleEvent)

	return router
}
