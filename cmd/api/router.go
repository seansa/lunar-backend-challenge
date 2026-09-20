package main

import (
	"github.com/gin-gonic/gin"
	"github.com/seansa/lunar-backend-challenge/internal/handler"
)

func newRouter(deps *dependencies) *gin.Engine {
	router := gin.New()

	handler := handler.New(nil)

	router.POST("/messages", handler.HandleEvent)

	return router
}
