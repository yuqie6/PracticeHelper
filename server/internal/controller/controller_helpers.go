package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/observability"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/service"
)

func requireStringID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if id == "" || len(id) > 64 || strings.ContainsAny(id, "/.\\") {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid id: %s", id))
		return "", false
	}
	return id, true
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
	type codedError interface {
		ErrorCode() string
	}
	var coded codedError
	if errors.As(err, &coded) && coded.ErrorCode() != "" {
		return coded.ErrorCode()
	}

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
	case errors.Is(err, service.ErrPromptSetNotFound):
		return "prompt_set_not_found"
	case errors.Is(err, service.ErrEmptyExportSelection):
		return "empty_export_selection"
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

func exportContentType(format string) string {
	switch format {
	case "json":
		return "application/json; charset=utf-8"
	case "pdf":
		return "application/pdf"
	default:
		return "text/markdown; charset=utf-8"
	}
}
