package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestExportSessionMarkdownIncludesTurnsAndReview(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedSessionForExport(t, store, "sess_export_with_review", true)
	svc := New(store, nil)

	filename, content, err := svc.ExportSession(context.Background(), session.ID, "markdown")
	if err != nil {
		t.Fatalf("ExportSession() error = %v", err)
	}

	if filename != "practicehelper-session-sess_export_with_review.md" {
		t.Fatalf("unexpected filename: %s", filename)
	}

	body := string(content)
	for _, snippet := range []string{
		"# PracticeHelper Session 报告",
		"- Session ID: sess_export_with_review",
		"- 绑定岗位: 后端工程师 - Example (Example)",
		"### 第 1 轮",
		"Redis 为什么快？",
		"缓存淘汰策略",
		"### 回答亮点",
		"举了缓存雪崩的真实场景",
		"### 检索轨迹",
		"memory_index_vector_rerank",
		"项目里的 Redis 观察",
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected export body to contain %q, got:\n%s", snippet, body)
		}
	}
}

func TestExportSessionMarkdownWithoutReviewShowsPendingNotice(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedSessionForExport(t, store, "sess_export_without_review", false)
	svc := New(store, nil)

	filename, content, err := svc.ExportSession(context.Background(), session.ID, "markdown")
	if err != nil {
		t.Fatalf("ExportSession() error = %v", err)
	}

	if filename != "practicehelper-session-sess_export_without_review.md" {
		t.Fatalf("unexpected filename: %s", filename)
	}

	body := string(content)
	if !strings.Contains(body, "> 复盘未生成。") {
		t.Fatalf("expected export body to mention missing review, got:\n%s", body)
	}
	if !strings.Contains(body, "### 第 2 轮") {
		t.Fatalf("expected export body to include follow-up turn, got:\n%s", body)
	}
}

func TestExportSessionJSONIncludesSessionAndReview(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedSessionForExport(t, store, "sess_export_json", true)
	loadedReview, err := store.GetReview(context.Background(), session.ReviewID)
	if err != nil {
		t.Fatalf("GetReview() error = %v", err)
	}
	if loadedReview == nil || loadedReview.RetrievalTrace == nil {
		t.Fatalf("expected retrieval trace on stored review, got %#v", loadedReview)
	}
	svc := New(store, nil)

	filename, content, err := svc.ExportSession(context.Background(), session.ID, "json")
	if err != nil {
		t.Fatalf("ExportSession() error = %v", err)
	}
	if filename != "practicehelper-session-sess_export_json.json" {
		t.Fatalf("unexpected filename: %s", filename)
	}

	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["format"] != "json" {
		t.Fatalf("unexpected format: %#v", payload["format"])
	}
	sessionPayload, ok := payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload, got %#v", payload["session"])
	}
	if sessionPayload["id"] != "sess_export_json" {
		t.Fatalf("unexpected session id: %#v", sessionPayload["id"])
	}
	if payload["review"] == nil {
		t.Fatalf("expected review payload, got %#v", payload["review"])
	}
	reviewPayload, ok := payload["review"].(map[string]any)
	if !ok {
		t.Fatalf("expected review payload map, got %#v", payload["review"])
	}
	if reviewPayload["retrieval_trace"] == nil {
		t.Fatalf("expected retrieval trace in review payload, got %#v", reviewPayload)
	}
}

func TestExportSessionPDFReturnsPDFBytes(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedSessionForExport(t, store, "sess_export_pdf", true)
	svc := New(store, nil)

	filename, content, err := svc.ExportSession(context.Background(), session.ID, "pdf")
	if err != nil {
		t.Fatalf("ExportSession() error = %v", err)
	}
	if filename != "practicehelper-session-sess_export_pdf.pdf" {
		t.Fatalf("unexpected filename: %s", filename)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-1.4")) {
		t.Fatalf("expected pdf header, got %q", content[:8])
	}
	if !bytes.Contains(content, []byte("STSong-Light")) {
		t.Fatalf("expected pdf font definition, got %q", content[:120])
	}
	if !bytes.Contains(content, []byte("<FEFF")) {
		t.Fatalf("expected utf-16 bom-prefixed text stream, got %q", content[:240])
	}
}

func TestExportSessionsZipIncludesSelectedMarkdownFiles(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	first := seedSessionForExport(t, store, "sess_export_batch_a", true)
	second := seedSessionForExport(t, store, "sess_export_batch_b", false)
	svc := New(store, nil)

	filename, content, err := svc.ExportSessions(
		context.Background(),
		[]string{first.ID, second.ID, first.ID},
		"markdown",
	)
	if err != nil {
		t.Fatalf("ExportSessions() error = %v", err)
	}

	if filename != "practicehelper-sessions-2-markdown.zip" {
		t.Fatalf("unexpected filename: %s", filename)
	}

	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("expected 2 files in zip, got %d", len(reader.File))
	}

	entries := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil {
			t.Fatalf("read zip entry %s: %v", file.Name, err)
		}
		entries[file.Name] = string(body)
	}

	firstBody := entries["practicehelper-session-sess_export_batch_a.md"]
	if firstBody == "" {
		t.Fatalf("missing first export entry: %v", reader.File)
	}
	if !strings.Contains(firstBody, "sess_export_batch_a") {
		t.Fatalf("unexpected first export content:\n%s", firstBody)
	}

	secondBody := entries["practicehelper-session-sess_export_batch_b.md"]
	if secondBody == "" {
		t.Fatalf("missing second export entry: %v", reader.File)
	}
	if !strings.Contains(secondBody, "> 复盘未生成。") {
		t.Fatalf("expected pending review notice in second export:\n%s", secondBody)
	}
}

func TestExportSessionsZipSupportsJSONEntries(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	first := seedSessionForExport(t, store, "sess_export_json_batch_a", true)
	second := seedSessionForExport(t, store, "sess_export_json_batch_b", false)
	svc := New(store, nil)

	filename, content, err := svc.ExportSessions(
		context.Background(),
		[]string{first.ID, second.ID},
		"json",
	)
	if err != nil {
		t.Fatalf("ExportSessions() error = %v", err)
	}
	if filename != "practicehelper-sessions-2-json.zip" {
		t.Fatalf("unexpected filename: %s", filename)
	}

	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("expected 2 files in zip, got %d", len(reader.File))
	}
	if !strings.HasSuffix(reader.File[0].Name, ".json") {
		t.Fatalf("expected json entry, got %s", reader.File[0].Name)
	}
}

func TestExportSessionsRejectsEmptySelection(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	svc := New(store, nil)

	_, _, err = svc.ExportSessions(context.Background(), nil, "markdown")
	if !errors.Is(err, ErrEmptyExportSelection) {
		t.Fatalf("expected ErrEmptyExportSelection, got %v", err)
	}
}

func TestExportSessionRejectsUnsupportedFormat(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedSessionForExport(t, store, "sess_export_invalid_format", true)
	svc := New(store, nil)

	_, _, err = svc.ExportSession(context.Background(), session.ID, "csv")
	if !errors.Is(err, ErrUnsupportedExportFormat) {
		t.Fatalf("expected ErrUnsupportedExportFormat, got %v", err)
	}
}
