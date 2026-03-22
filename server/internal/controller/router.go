package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/service"
)

type Handler struct {
	service *service.Service
}

func NewRouter(svc *service.Service) *gin.Engine {
	return NewRouterWithInternalToken(svc, "")
}

func NewRouterWithInternalToken(svc *service.Service, internalToken string) *gin.Engine {
	handler := &Handler{service: svc}

	router := gin.New()
	router.Use(gin.Recovery(), requestLogger(), corsMiddleware())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	{
		registerProfileRoutes(api, handler)
		registerJobTargetRoutes(api, handler)
		registerProjectRoutes(api, handler)
		registerPromptRoutes(api, handler)
		registerSessionRoutes(api, handler)
	}

	internal := router.Group("/internal")
	internal.Use(internalAuthMiddleware(internalToken))
	{
		registerInternalRoutes(internal, handler)
	}

	return router
}
