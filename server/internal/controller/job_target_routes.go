package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func registerJobTargetRoutes(api *gin.RouterGroup, handler *Handler) {
	api.GET("/job-targets", handler.listJobTargets)
	api.POST("/job-targets", handler.createJobTarget)
	api.POST("/job-targets/clear-active", handler.clearActiveJobTarget)
	api.GET("/job-targets/analysis-runs/:id", handler.getJobTargetAnalysisRun)
	api.GET("/job-targets/:id", handler.getJobTarget)
	api.PATCH("/job-targets/:id", handler.updateJobTarget)
	api.POST("/job-targets/:id/activate", handler.activateJobTarget)
	api.POST("/job-targets/:id/analyze", handler.analyzeJobTarget)
	api.GET("/job-targets/:id/analysis-runs", handler.listJobTargetAnalysisRuns)
}

func (h *Handler) listJobTargets(c *gin.Context) {
	data, err := h.service.ListJobTargets(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) createJobTarget(c *gin.Context) {
	var request domain.JobTargetInput
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.CreateJobTarget(c.Request.Context(), request)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func (h *Handler) getJobTarget(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.GetJobTarget(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobTargetNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) updateJobTarget(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	var request domain.JobTargetInput
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.UpdateJobTarget(c.Request.Context(), id, request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobTargetNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) analyzeJobTarget(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.AnalyzeJobTarget(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobTargetNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusBadGateway, err)
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func (h *Handler) activateJobTarget(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.ActivateJobTarget(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobTargetNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) clearActiveJobTarget(c *gin.Context) {
	data, err := h.service.ClearActiveJobTarget(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) listJobTargetAnalysisRuns(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.ListJobTargetAnalysisRuns(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobTargetNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getJobTargetAnalysisRun(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.GetJobTargetAnalysisRun(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobTargetAnalysisNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}
