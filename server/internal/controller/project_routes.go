package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/service"
)

func registerProjectRoutes(api *gin.RouterGroup, handler *Handler) {
	api.POST("/projects/import", handler.importProject)
	api.GET("/projects", handler.listProjects)
	api.GET("/projects/:id", handler.getProject)
	api.PATCH("/projects/:id", handler.updateProject)
	api.GET("/import-jobs", handler.listImportJobs)
	api.GET("/import-jobs/:id", handler.getImportJob)
	api.POST("/import-jobs/:id/retry", handler.retryImportJob)
}

func (h *Handler) importProject(c *gin.Context) {
	var request domain.ProjectImportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.ImportProject(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrAlreadyImported):
			writeError(c, http.StatusConflict, err)
		default:
			writeError(c, http.StatusBadGateway, err)
		}
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": data})
}

func (h *Handler) listProjects(c *gin.Context) {
	data, err := h.service.ListProjects(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getProject(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.GetProject(c.Request.Context(), id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		writeError(c, http.StatusNotFound, errors.New("project not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) updateProject(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	var request domain.ProjectProfileInput
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.UpdateProject(c.Request.Context(), id, request)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) listImportJobs(c *gin.Context) {
	data, err := h.service.ListProjectImportJobs(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getImportJob(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.GetProjectImportJob(c.Request.Context(), id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		writeError(c, http.StatusNotFound, errors.New("import job not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) retryImportJob(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.RetryProjectImportJob(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrImportJobNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusBadGateway, err)
		}
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"data": data})
}
