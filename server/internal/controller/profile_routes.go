package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/domain"
)

func registerProfileRoutes(api *gin.RouterGroup, handler *Handler) {
	api.GET("/dashboard", handler.getDashboard)

	api.GET("/profile", handler.getProfile)
	api.POST("/profile", handler.saveProfile)
	api.PATCH("/profile", handler.saveProfile)
}

func (h *Handler) getDashboard(c *gin.Context) {
	data, err := h.service.GetDashboard(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getProfile(c *gin.Context) {
	data, err := h.service.GetProfile(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) saveProfile(c *gin.Context) {
	var request domain.UserProfileInput
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.SaveProfile(c.Request.Context(), request)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}
