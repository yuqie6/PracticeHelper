package controller

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/service"
)

const internalTokenHeader = "X-PracticeHelper-Internal-Token"

func registerInternalRoutes(internal *gin.RouterGroup, handler *Handler) {
	internal.GET("/search-chunks", handler.searchChunksInternal)
	internal.GET("/session-detail/:id", handler.getSessionDetailInternal)
}

func (h *Handler) searchChunksInternal(c *gin.Context) {
	projectID := c.Query("project_id")
	query := c.Query("query")
	if projectID == "" || query == "" {
		writeError(c, http.StatusBadRequest, errors.New("project_id and query are required"))
		return
	}

	limit := 6
	if rawLimit := c.Query("limit"); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value <= 0 || value > 20 {
			writeError(c, http.StatusBadRequest, errors.New("limit must be between 1 and 20"))
			return
		}
		limit = value
	}

	data, err := h.service.SearchProjectChunksForAgent(c.Request.Context(), projectID, query, limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getSessionDetailInternal(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.GetAgentSessionDetail(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSessionNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func internalAuthMiddleware(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expectedToken == "" {
			writeError(c, http.StatusServiceUnavailable, errors.New("internal token is not configured"))
			c.Abort()
			return
		}

		actualToken := c.GetHeader(internalTokenHeader)
		if subtle.ConstantTimeCompare([]byte(actualToken), []byte(expectedToken)) != 1 {
			writeError(c, http.StatusUnauthorized, errors.New("invalid internal token"))
			c.Abort()
			return
		}

		c.Next()
	}
}
