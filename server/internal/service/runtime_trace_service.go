package service

import (
	"fmt"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

func latencyMsSince(startedAt time.Time) float64 {
	return float64(time.Since(startedAt).Microseconds()) / 1000
}

func appendPromptTrace(
	meta *domain.PromptExecutionMeta,
	emit func(domain.StreamEvent) error,
	flowName string,
	phase string,
	status string,
	code string,
	message string,
	details map[string]any,
) {
	if meta == nil {
		return
	}
	if meta.RuntimeTrace == nil {
		meta.RuntimeTrace = &domain.RuntimeTrace{}
	}

	entry := domain.RuntimeTraceEntry{
		Flow:    strings.TrimSuffix(flowName, "_stream"),
		Phase:   phase,
		Status:  status,
		Code:    code,
		Message: message,
		Details: cloneTraceDetails(details),
	}
	meta.RuntimeTrace.Entries = append(meta.RuntimeTrace.Entries, entry)

	if emit != nil {
		_ = emit(domain.StreamEvent{Type: "trace", Data: entry})
	}
}

func appendPersistFailureTrace(
	meta *domain.PromptExecutionMeta,
	emit func(domain.StreamEvent) error,
	flowName string,
	section string,
	err error,
) {
	appendPromptTrace(
		meta,
		emit,
		flowName,
		"persist",
		"error",
		"persist_failed",
		fmt.Sprintf("%s 持久化失败：%v", section, err),
		map[string]any{
			"section": section,
		},
	)
}

func cloneTraceDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(details))
	for key, value := range details {
		cloned[key] = value
	}
	return cloned
}
