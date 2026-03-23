package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func registerSessionRoutes(api *gin.RouterGroup, handler *Handler) {
	api.GET("/sessions", handler.listSessions)
	api.POST("/sessions", handler.createSession)
	api.POST("/sessions/export", handler.exportSessions)
	api.POST("/sessions/stream", handler.createSessionStream)
	api.GET("/sessions/:id", handler.getSession)
	api.GET("/sessions/:id/evaluation-logs", handler.listSessionEvaluationLogs)
	api.GET("/sessions/:id/export", handler.exportSession)
	api.POST("/sessions/:id/answer", handler.submitAnswer)
	api.POST("/sessions/:id/answer/stream", handler.submitAnswerStream)
	api.POST("/sessions/:id/retry-review", handler.retrySessionReview)

	api.GET("/reviews/:id", handler.getReview)
	api.GET("/weaknesses", handler.listWeaknesses)
	api.GET("/weaknesses/trends", handler.getWeaknessTrends)
	api.GET("/reviews/due", handler.listDueReviews)
	api.POST("/reviews/due/:id/complete", handler.completeDueReview)
}

func (h *Handler) createSession(c *gin.Context) {
	var request domain.CreateSessionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.CreateSession(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidMode):
			writeError(c, http.StatusBadRequest, err)
		case errors.Is(err, service.ErrInvalidPromptOverlay):
			writeError(c, http.StatusBadRequest, err)
		case errors.Is(err, service.ErrProjectNotFound):
			writeError(c, http.StatusNotFound, err)
		case errors.Is(err, service.ErrJobTargetNotFound):
			writeError(c, http.StatusNotFound, err)
		case errors.Is(err, service.ErrJobTargetNotReady):
			writeError(c, http.StatusConflict, err)
		default:
			writeError(c, http.StatusBadGateway, err)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func (h *Handler) createSessionStream(c *gin.Context) {
	var request domain.CreateSessionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	streamJSON(c, http.StatusCreated, func(emit func(domain.StreamEvent) error) (any, error) {
		return h.service.CreateSessionStream(c.Request.Context(), request, emit)
	})
}

func (h *Handler) listSessions(c *gin.Context) {
	var req domain.ListSessionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	data, err := h.service.ListSessions(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getSession(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.GetSession(c.Request.Context(), id)
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

func (h *Handler) listSessionEvaluationLogs(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.ListSessionEvaluationLogs(c.Request.Context(), id)
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

func (h *Handler) exportSession(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	format := c.Query("format")
	filename, content, err := h.service.ExportSession(
		c.Request.Context(),
		id,
		format,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnsupportedExportFormat):
			writeError(c, http.StatusBadRequest, err)
		case errors.Is(err, service.ErrSessionNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}

	c.Header("Content-Type", exportContentType(format))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, exportContentType(format), content)
}

func (h *Handler) exportSessions(c *gin.Context) {
	var request domain.ExportSessionsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	filename, content, err := h.service.ExportSessions(
		c.Request.Context(),
		request.SessionIDs,
		request.Format,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyExportSelection),
			errors.Is(err, service.ErrUnsupportedExportFormat):
			writeError(c, http.StatusBadRequest, err)
		case errors.Is(err, service.ErrSessionNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/zip", content)
}

func (h *Handler) submitAnswer(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	var request domain.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.SubmitAnswer(c.Request.Context(), id, request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSessionNotFound):
			writeError(c, http.StatusNotFound, err)
		case errors.Is(err, service.ErrSessionBusy),
			errors.Is(err, service.ErrSessionReviewPending),
			errors.Is(err, service.ErrSessionCompleted),
			errors.Is(err, service.ErrSessionAnswerConflict):
			writeError(c, http.StatusConflict, err)
		case errors.Is(err, service.ErrReviewGenerationRetry):
			writeError(c, http.StatusBadGateway, err)
		default:
			writeError(c, http.StatusBadGateway, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) submitAnswerStream(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	var request domain.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	streamJSON(c, http.StatusOK, func(emit func(domain.StreamEvent) error) (any, error) {
		return h.service.SubmitAnswerStream(c.Request.Context(), id, request, emit)
	})
}

func (h *Handler) retrySessionReview(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.RetrySessionReview(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSessionNotFound):
			writeError(c, http.StatusNotFound, err)
		case errors.Is(err, service.ErrSessionNotRecoverable),
			errors.Is(err, service.ErrSessionBusy),
			errors.Is(err, service.ErrSessionCompleted):
			writeError(c, http.StatusConflict, err)
		case errors.Is(err, service.ErrReviewGenerationRetry):
			writeError(c, http.StatusBadGateway, err)
		default:
			writeError(c, http.StatusBadGateway, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getReview(c *gin.Context) {
	id, ok := requireStringID(c)
	if !ok {
		return
	}
	data, err := h.service.GetReview(c.Request.Context(), id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		writeError(c, http.StatusNotFound, errors.New("review not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) listWeaknesses(c *gin.Context) {
	data, err := h.service.ListWeaknesses(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getWeaknessTrends(c *gin.Context) {
	data, err := h.service.GetWeaknessTrends(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) listDueReviews(c *gin.Context) {
	data, err := h.service.ListDueReviews(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) completeDueReview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid id: %s", idStr))
		return
	}
	if err := h.service.CompleteDueReview(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, service.ErrReviewScheduleNotFound), errors.Is(err, service.ErrSessionNotFound):
			writeError(c, http.StatusNotFound, err)
		default:
			writeError(c, http.StatusInternalServerError, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}
