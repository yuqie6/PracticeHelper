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
