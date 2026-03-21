package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/observability"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/service"
)

type Handler struct {
	service *service.Service
}

func NewRouter(svc *service.Service) *gin.Engine {
	handler := &Handler{service: svc}

	router := gin.New()
	router.Use(gin.Recovery(), requestLogger(), corsMiddleware())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	{
		api.GET("/dashboard", handler.getDashboard)

		api.GET("/profile", handler.getProfile)
		api.POST("/profile", handler.saveProfile)
		api.PATCH("/profile", handler.saveProfile)

		api.GET("/job-targets", handler.listJobTargets)
		api.POST("/job-targets", handler.createJobTarget)
		api.POST("/job-targets/clear-active", handler.clearActiveJobTarget)
		api.GET("/job-targets/analysis-runs/:id", handler.getJobTargetAnalysisRun)
		api.GET("/job-targets/:id", handler.getJobTarget)
		api.PATCH("/job-targets/:id", handler.updateJobTarget)
		api.POST("/job-targets/:id/activate", handler.activateJobTarget)
		api.POST("/job-targets/:id/analyze", handler.analyzeJobTarget)
		api.GET("/job-targets/:id/analysis-runs", handler.listJobTargetAnalysisRuns)

		api.POST("/projects/import", handler.importProject)
		api.GET("/projects", handler.listProjects)
		api.GET("/projects/:id", handler.getProject)
		api.PATCH("/projects/:id", handler.updateProject)
		api.GET("/import-jobs", handler.listImportJobs)
		api.GET("/import-jobs/:id", handler.getImportJob)
		api.POST("/import-jobs/:id/retry", handler.retryImportJob)

		api.GET("/sessions", handler.listSessions)
		api.POST("/sessions", handler.createSession)
		api.POST("/sessions/stream", handler.createSessionStream)
		api.GET("/sessions/:id", handler.getSession)
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

	return router
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
	data, err := h.service.GetJobTarget(c.Request.Context(), c.Param("id"))
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
	var request domain.JobTargetInput
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.UpdateJobTarget(c.Request.Context(), c.Param("id"), request)
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
	data, err := h.service.AnalyzeJobTarget(c.Request.Context(), c.Param("id"))
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
	data, err := h.service.ActivateJobTarget(c.Request.Context(), c.Param("id"))
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
	data, err := h.service.ListJobTargetAnalysisRuns(c.Request.Context(), c.Param("id"))
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
	data, err := h.service.GetJobTargetAnalysisRun(c.Request.Context(), c.Param("id"))
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
	data, err := h.service.GetProject(c.Request.Context(), c.Param("id"))
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
	var request domain.ProjectProfileInput
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.UpdateProject(c.Request.Context(), c.Param("id"), request)
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
	data, err := h.service.GetProjectImportJob(c.Request.Context(), c.Param("id"))
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
	data, err := h.service.RetryProjectImportJob(c.Request.Context(), c.Param("id"))
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
	data, err := h.service.GetSession(c.Request.Context(), c.Param("id"))
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
	filename, content, err := h.service.ExportSession(
		c.Request.Context(),
		c.Param("id"),
		c.Query("format"),
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

	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", content)
}

func (h *Handler) submitAnswer(c *gin.Context) {
	var request domain.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	data, err := h.service.SubmitAnswer(c.Request.Context(), c.Param("id"), request)
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
	var request domain.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	streamJSON(c, http.StatusOK, func(emit func(domain.StreamEvent) error) (any, error) {
		return h.service.SubmitAnswerStream(c.Request.Context(), c.Param("id"), request, emit)
	})
}

func (h *Handler) retrySessionReview(c *gin.Context) {
	data, err := h.service.RetrySessionReview(c.Request.Context(), c.Param("id"))
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
	data, err := h.service.GetReview(c.Request.Context(), c.Param("id"))
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

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(observability.RequestIDHeader)
		if requestID == "" {
			requestID = observability.NewRequestID()
		}

		startedAt := time.Now()
		c.Writer.Header().Set(observability.RequestIDHeader, requestID)
		c.Request = c.Request.WithContext(observability.WithRequestID(c.Request.Context(), requestID))

		c.Next()

		duration := time.Since(startedAt)
		attrs := []any{
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.FullPath(),
			"raw_path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"client_ip", c.ClientIP(),
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
			slog.ErrorContext(c.Request.Context(), "http request failed", attrs...)
			return
		}

		slog.InfoContext(c.Request.Context(), "http request completed", attrs...)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func writeError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    errorCode(err),
			"message": err.Error(),
		},
	})
}

func streamJSON(
	c *gin.Context,
	status int,
	run func(emit func(domain.StreamEvent) error) (any, error),
) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		writeError(c, http.StatusInternalServerError, errors.New("streaming is not supported"))
		return
	}

	c.Status(status)
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	encoder := json.NewEncoder(c.Writer)
	emit := func(event domain.StreamEvent) error {
		if err := encoder.Encode(event); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	result, err := run(emit)
	if err != nil {
		_ = emit(domain.StreamEvent{Type: "error", Code: errorCode(err), Message: err.Error()})
		return
	}

	_ = emit(domain.StreamEvent{Type: "result", Data: result})
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, service.ErrInvalidMode):
		return "invalid_mode"
	case errors.Is(err, service.ErrProjectNotFound):
		return "project_not_found"
	case errors.Is(err, service.ErrJobTargetNotFound):
		return "job_target_not_found"
	case errors.Is(err, service.ErrJobTargetNotReady):
		return "job_target_not_ready"
	case errors.Is(err, service.ErrJobTargetAnalysisNotFound):
		return "job_target_analysis_not_found"
	case errors.Is(err, service.ErrSessionNotFound):
		return "session_not_found"
	case errors.Is(err, service.ErrUnsupportedExportFormat):
		return "invalid_export_format"
	case errors.Is(err, service.ErrImportJobNotFound):
		return "import_job_not_found"
	case errors.Is(err, service.ErrSessionNotRecoverable):
		return "session_not_recoverable"
	case errors.Is(err, service.ErrSessionBusy):
		return "session_busy"
	case errors.Is(err, service.ErrSessionReviewPending):
		return "session_review_pending"
	case errors.Is(err, service.ErrSessionCompleted):
		return "session_completed"
	case errors.Is(err, service.ErrSessionAnswerConflict):
		return "session_answer_conflict"
	case errors.Is(err, service.ErrReviewGenerationRetry):
		return "review_generation_retry"
	case errors.Is(err, repo.ErrAlreadyImported):
		return "project_already_imported"
	default:
		return "unknown_error"
	}
}
