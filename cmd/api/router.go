package main

import (
	"github.com/gin-gonic/gin"
	"github.com/seansa/lunar-backend-challenge/internal/handler"
)

func newRouter(deps *dependencies) *gin.Engine {
	router := gin.New()

	handler := handler.New(deps.service, deps.consumer, deps.service, deps.service)

	router.POST("/messages", handler.Process)
	router.POST("/events", handler.HandleEvent)

	router.GET("/rockets", handler.ListRockets)
	router.GET("/rockets/:channel", handler.GetRocket)
	router.GET("/rockets/:channel/events", handler.ListEvents)


	return router
}
