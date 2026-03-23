package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func registerPromptRoutes(api *gin.RouterGroup, handler *Handler) {
	api.GET("/prompt-sets", handler.listPromptSets)
	api.GET("/prompt-preferences", handler.getPromptPreferences)
	api.PATCH("/prompt-preferences", handler.savePromptPreferences)
	api.GET("/prompt-experiments/prompt-sets", handler.listPromptExperimentPromptSets)
	api.GET("/prompt-experiments", handler.getPromptExperiment)
}

func (h *Handler) listPromptSets(c *gin.Context) {
	data, err := h.service.ListPromptSets(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) listPromptExperimentPromptSets(c *gin.Context) {
	data, err := h.service.ListPromptExperimentPromptSets(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getPromptPreferences(c *gin.Context) {
	data, err := h.service.GetPromptPreferences(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) savePromptPreferences(c *gin.Context) {
	var request domain.PromptOverlay
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.SavePromptPreferences(c.Request.Context(), &request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPromptOverlay):
			writeError(c, http.StatusBadRequest, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getPromptExperiment(c *gin.Context) {
	var req domain.PromptExperimentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.GetPromptExperiment(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPromptSetNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusBadRequest, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
