package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"practicehelper/server/internal/domain"
)

const (
	sessionExportFormatMarkdown = "markdown"
	sessionExportFormatJSON     = "json"
	sessionExportFormatPDF      = "pdf"
)

func (s *Service) ExportSession(
	ctx context.Context,
	sessionID string,
	format string,
) (string, []byte, error) {
	if !isSupportedSessionExportFormat(format) {
		return "", nil, ErrUnsupportedExportFormat
	}

	return s.buildSessionExport(ctx, sessionID, format)
}

func (s *Service) ExportSessions(
	ctx context.Context,
	sessionIDs []string,
	format string,
) (string, []byte, error) {
	if !isSupportedSessionExportFormat(format) {
		return "", nil, ErrUnsupportedExportFormat
	}

	uniqueIDs := uniqueSessionIDs(sessionIDs)
	if len(uniqueIDs) == 0 {
		return "", nil, ErrEmptyExportSelection
	}

	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)

	for _, sessionID := range uniqueIDs {
		filename, content, err := s.buildSessionExport(ctx, sessionID, format)
		if err != nil {
			return "", nil, err
		}

		entry, err := writer.Create(filename)
		if err != nil {
			return "", nil, fmt.Errorf("create zip entry: %w", err)
		}
		if _, err := entry.Write(content); err != nil {
			return "", nil, fmt.Errorf("write zip entry: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", nil, fmt.Errorf("close zip archive: %w", err)
	}

	filename := fmt.Sprintf(
		"practicehelper-sessions-%d-%s.zip",
		len(uniqueIDs),
		format,
	)
	return filename, archive.Bytes(), nil
}

func (s *Service) buildSessionExport(
	ctx context.Context,
	sessionID string,
	format string,
) (string, []byte, error) {
	session, review, err := s.loadSessionExportData(ctx, sessionID)
	if err != nil {
		return "", nil, err
	}

	content, err := renderSessionExportContent(session, review, format)
	if err != nil {
		return "", nil, err
	}

	return exportFilename(session.ID, format), content, nil
}

func uniqueSessionIDs(sessionIDs []string) []string {
	seen := make(map[string]struct{}, len(sessionIDs))
	items := make([]string, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		id := strings.TrimSpace(sessionID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		items = append(items, id)
	}
	return items
}

func isSupportedSessionExportFormat(format string) bool {
	switch format {
	case sessionExportFormatMarkdown, sessionExportFormatJSON, sessionExportFormatPDF:
		return true
	default:
		return false
	}
}

func exportFilename(sessionID string, format string) string {
	return fmt.Sprintf("practicehelper-session-%s.%s", sessionID, exportExtension(format))
}

func exportExtension(format string) string {
	switch format {
	case sessionExportFormatJSON:
		return "json"
	case sessionExportFormatPDF:
		return "pdf"
	default:
		return "md"
	}
}

func (s *Service) loadSessionExportData(
	ctx context.Context,
	sessionID string,
) (*domain.TrainingSession, *domain.ReviewCard, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, ErrSessionNotFound
	}

	var review *domain.ReviewCard
	if session.ReviewID != "" {
		review, err = s.repo.GetReview(ctx, session.ReviewID)
		if err != nil {
			return nil, nil, err
		}
	}

	return session, review, nil
}

func renderSessionExportContent(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	format string,
) ([]byte, error) {
	switch format {
	case sessionExportFormatMarkdown:
		return []byte(renderSessionMarkdown(session, review)), nil
	case sessionExportFormatJSON:
		return renderSessionJSON(session, review)
	case sessionExportFormatPDF:
		return renderSessionPDF(session, review)
	default:
		return nil, ErrUnsupportedExportFormat
	}
}

type sessionExportEnvelope struct {
	GeneratedAt string                  `json:"generated_at"`
	Format      string                  `json:"format"`
	Session     *domain.TrainingSession `json:"session"`
	Review      *domain.ReviewCard      `json:"review,omitempty"`
}

func renderSessionJSON(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
) ([]byte, error) {
	payload := sessionExportEnvelope{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Format:      sessionExportFormatJSON,
		Session:     session,
		Review:      review,
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal session export json: %w", err)
	}
	return append(content, '\n'), nil
}

func renderSessionPDF(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
) ([]byte, error) {
	source := renderSessionMarkdown(session, review)
	return buildTextPDF(source), nil
}

func buildTextPDF(text string) []byte {
	const (
		pageWidth       = 595
		pageHeight      = 842
		fontSize        = 11
		lineHeight      = 14
		startX          = 40
		startY          = 798
		maxLinesPerPage = 48
	)

	lines := wrapPDFLines(text, 42)
	if len(lines) == 0 {
		lines = []string{" "}
	}

	pages := chunkPDFLines(lines, maxLinesPerPage)
	pageObjectIDs := make([]int, 0, len(pages))
	contentObjectIDs := make([]int, 0, len(pages))
	for index := range pages {
		pageObjectIDs = append(pageObjectIDs, 5+index*2)
		contentObjectIDs = append(contentObjectIDs, 6+index*2)
	}

	kids := make([]string, 0, len(pageObjectIDs))
	objects := make(map[int]string, 4+len(pages)*2)
	for _, id := range pageObjectIDs {
		kids = append(kids, fmt.Sprintf("%d 0 R", id))
	}

	objects[1] = "<< /Type /Catalog /Pages 2 0 R >>"
	objects[2] = fmt.Sprintf(
		"<< /Type /Pages /Count %d /Kids [%s] >>",
		len(pageObjectIDs),
		strings.Join(kids, " "),
	)
	objects[3] = "<< /Type /Font /Subtype /Type0 /BaseFont /STSong-Light /Encoding /UniGB-UCS2-H /DescendantFonts [4 0 R] >>"
	objects[4] = "<< /Type /Font /Subtype /CIDFontType0 /BaseFont /STSong-Light /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 4 >> /DW 1000 >>"

	for index, pageLines := range pages {
		content := buildPDFPageContent(pageLines, fontSize, lineHeight, startX, startY)
		objects[pageObjectIDs[index]] = fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>",
			pageWidth,
			pageHeight,
			contentObjectIDs[index],
		)
		objects[contentObjectIDs[index]] = fmt.Sprintf(
			"<< /Length %d >>\nstream\n%s\nendstream",
			len(content),
			content,
		)
	}

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n%\xe4\xbd\xa0\xe5\xa5\xbd\n")

	offsets := make([]int, 0, len(objects)+1)
	offsets = append(offsets, 0)
	for objectID := 1; objectID <= len(objects); objectID++ {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", objectID, objects[objectID])
	}

	xrefOffset := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n", len(objects)+1)
	pdf.WriteString("0000000000 65535 f \n")
	for objectID := 1; objectID <= len(objects); objectID++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[objectID])
	}

	fmt.Fprintf(
		&pdf,
		"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF",
		len(objects)+1,
		xrefOffset,
	)

	return pdf.Bytes()
}

func buildPDFPageContent(
	lines []string,
	fontSize int,
	lineHeight int,
	startX int,
	startY int,
) string {
	var builder strings.Builder
	builder.WriteString("BT\n")
	fmt.Fprintf(&builder, "/F1 %d Tf\n", fontSize)
	fmt.Fprintf(&builder, "%d TL\n", lineHeight)
	fmt.Fprintf(&builder, "%d %d Td\n", startX, startY)
	for index, line := range lines {
		if index > 0 {
			builder.WriteString("T*\n")
		}
		builder.WriteString("<")
		builder.WriteString(encodePDFText(line))
		builder.WriteString("> Tj\n")
	}
	builder.WriteString("ET")
	return builder.String()
}

func wrapPDFLines(text string, maxRunes int) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))

	for _, raw := range rawLines {
		runes := []rune(strings.TrimRight(raw, " "))
		if len(runes) == 0 {
			lines = append(lines, " ")
			continue
		}

		for len(runes) > maxRunes {
			split := bestPDFSplit(runes, maxRunes)
			chunk := strings.TrimSpace(string(runes[:split]))
			if chunk == "" {
				chunk = string(runes[:split])
			}
			lines = append(lines, chunk)
			runes = []rune(strings.TrimLeft(string(runes[split:]), " "))
		}

		if len(runes) > 0 {
			lines = append(lines, string(runes))
		}
	}

	return lines
}

func bestPDFSplit(runes []rune, maxRunes int) int {
	if len(runes) <= maxRunes {
		return len(runes)
	}

	for index := maxRunes; index > maxRunes/2; index-- {
		if runes[index] == ' ' || runes[index] == '\t' {
			return index
		}
	}
	return maxRunes
}

func chunkPDFLines(lines []string, maxLines int) [][]string {
	pages := make([][]string, 0, (len(lines)/maxLines)+1)
	for start := 0; start < len(lines); start += maxLines {
		end := start + maxLines
		if end > len(lines) {
			end = len(lines)
		}
		pages = append(pages, lines[start:end])
	}
	return pages
}

func encodePDFText(text string) string {
	if text == "" {
		text = " "
	}

	encodedRunes := utf16.Encode([]rune(text))
	bytes := make([]byte, 0, len(encodedRunes)*2)
	for _, value := range encodedRunes {
		bytes = append(bytes, byte(value>>8), byte(value))
	}
	return strings.ToUpper(hex.EncodeToString(bytes))
}

func renderSessionMarkdown(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
) string {
	var builder strings.Builder

	builder.WriteString("# PracticeHelper Session 报告\n\n")
	builder.WriteString("## 基本信息\n\n")
	fmt.Fprintf(&builder, "- Session ID: %s\n", session.ID)
	fmt.Fprintf(&builder, "- 模式: %s\n", session.Mode)
	fmt.Fprintf(&builder, "- 主题: %s\n", fallbackText(session.Topic))
	fmt.Fprintf(&builder, "- 强度: %s\n", fallbackText(session.Intensity))
	fmt.Fprintf(&builder, "- 状态: %s\n", session.Status)
	fmt.Fprintf(&builder, "- 最大轮次: %d\n", session.MaxTurns)
	fmt.Fprintf(&builder, "- 总分: %s\n", formatScore(session.TotalScore))
	fmt.Fprintf(&builder, "- 创建时间: %s\n", formatTimeValue(&session.CreatedAt))
	fmt.Fprintf(&builder, "- 开始时间: %s\n", formatTimeValue(session.StartedAt))
	fmt.Fprintf(&builder, "- 结束时间: %s\n", formatTimeValue(session.EndedAt))
	fmt.Fprintf(&builder, "- 绑定岗位: %s\n", formatJobTarget(session.JobTarget))
	fmt.Fprintf(&builder, "- 绑定项目: %s\n", formatProject(session.Project))
	builder.WriteString("\n")

	builder.WriteString("## 训练轮次\n\n")
	if len(session.Turns) == 0 {
		builder.WriteString("> 当前还没有题目或作答记录。\n\n")
	} else {
		for _, turn := range session.Turns {
			fmt.Fprintf(&builder, "### 第 %d 轮\n\n", turn.TurnIndex)
			builder.WriteString("#### 问题\n\n")
			builder.WriteString(fallbackText(turn.Question))
			builder.WriteString("\n\n")

			builder.WriteString("#### 预期要点\n\n")
			writeList(&builder, turn.ExpectedPoints, "当前没有记录预期要点。")

			builder.WriteString("#### 回答\n\n")
			writeBlock(&builder, turn.Answer, "当前还没有回答内容。")

			builder.WriteString("#### 评估\n\n")
			if turn.Evaluation == nil {
				builder.WriteString("> 当前还没有评估结果。\n\n")
			} else {
				fmt.Fprintf(&builder, "- 得分: %s\n", formatScore(turn.Evaluation.Score))
				fmt.Fprintf(&builder, "- 总结: %s\n", fallbackText(turn.Evaluation.Headline))
				fmt.Fprintf(&builder, "- 建议: %s\n", fallbackText(turn.Evaluation.Suggestion))
				fmt.Fprintf(&builder, "- 追问意图: %s\n", fallbackText(turn.Evaluation.FollowupIntent))
				fmt.Fprintf(&builder, "- 下一刀追问: %s\n\n", fallbackText(turn.Evaluation.FollowupQuestion))

				builder.WriteString("##### 优点\n\n")
				writeList(&builder, turn.Evaluation.Strengths, "当前没有记录优点。")

				builder.WriteString("##### 待补强点\n\n")
				writeList(&builder, turn.Evaluation.Gaps, "当前没有记录待补强点。")

				builder.WriteString("##### 分项得分\n\n")
				writeScoreBreakdown(
					&builder,
					turn.Evaluation.ScoreBreakdown,
					"当前没有记录分项得分。",
				)

				builder.WriteString("##### 识别到的薄弱点\n\n")
				writeWeaknessHits(
					&builder,
					turn.Evaluation.WeaknessHits,
					"当前没有识别到薄弱点。",
				)
			}
		}
	}

	builder.WriteString("## 最终复盘\n\n")
	if review == nil {
		builder.WriteString("> 复盘未生成。\n")
		return builder.String()
	}

	fmt.Fprintf(&builder, "- 复盘 ID: %s\n", review.ID)
	fmt.Fprintf(&builder, "- 生成时间: %s\n\n", formatTimeValue(&review.CreatedAt))

	builder.WriteString("### 总结\n\n")
	builder.WriteString(fallbackText(review.Overall))
	builder.WriteString("\n\n")

	builder.WriteString("### 最该优先修正\n\n")
	fmt.Fprintf(&builder, "- 问题: %s\n", fallbackText(review.TopFix))
	fmt.Fprintf(&builder, "- 原因: %s\n\n", fallbackText(review.TopFixReason))

	builder.WriteString("### 回答亮点\n\n")
	writeList(&builder, review.Highlights, "当前没有记录亮点。")

	builder.WriteString("### 需要补强\n\n")
	writeList(&builder, review.Gaps, "当前没有记录补强项。")

	builder.WriteString("### 推荐练习主题\n\n")
	writeList(&builder, review.SuggestedTopics, "当前没有推荐主题。")

	builder.WriteString("### 下一轮重点\n\n")
	writeList(&builder, review.NextTrainingFocus, "当前没有下一轮重点。")

	builder.WriteString("### 推荐下一轮\n\n")
	if review.RecommendedNext == nil {
		builder.WriteString("> 当前没有推荐下一轮。\n\n")
	} else {
		fmt.Fprintf(&builder, "- 模式: %s\n", review.RecommendedNext.Mode)
		fmt.Fprintf(&builder, "- 主题: %s\n", fallbackText(review.RecommendedNext.Topic))
		fmt.Fprintf(&builder, "- 项目 ID: %s\n", fallbackText(review.RecommendedNext.ProjectID))
		fmt.Fprintf(&builder, "- 原因: %s\n\n", fallbackText(review.RecommendedNext.Reason))
	}

	builder.WriteString("### 复盘分项得分\n\n")
	writeScoreBreakdown(&builder, review.ScoreBreakdown, "当前没有复盘分项得分。")

	return builder.String()
}

func writeList(builder *strings.Builder, items []string, empty string) {
	if len(items) == 0 {
		fmt.Fprintf(builder, "> %s\n\n", empty)
		return
	}

	for _, item := range items {
		fmt.Fprintf(builder, "- %s\n", fallbackText(item))
	}
	builder.WriteString("\n")
}

func writeBlock(builder *strings.Builder, content string, empty string) {
	text := strings.TrimSpace(content)
	if text == "" {
		fmt.Fprintf(builder, "> %s\n\n", empty)
		return
	}

	builder.WriteString("````text\n")
	builder.WriteString(text)
	builder.WriteString("\n````\n\n")
}

func writeScoreBreakdown(
	builder *strings.Builder,
	breakdown map[string]float64,
	empty string,
) {
	if len(breakdown) == 0 {
		fmt.Fprintf(builder, "> %s\n\n", empty)
		return
	}

	keys := make([]string, 0, len(breakdown))
	for key := range breakdown {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Fprintf(builder, "- %s: %s\n", key, formatScore(breakdown[key]))
	}
	builder.WriteString("\n")
}

func writeWeaknessHits(
	builder *strings.Builder,
	items []domain.WeaknessHit,
	empty string,
) {
	if len(items) == 0 {
		fmt.Fprintf(builder, "> %s\n\n", empty)
		return
	}

	for _, item := range items {
		fmt.Fprintf(
			builder,
			"- %s / %s / 严重度 %s\n",
			fallbackText(item.Kind),
			fallbackText(item.Label),
			formatScore(item.Severity),
		)
	}
	builder.WriteString("\n")
}

func fallbackText(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return "—"
	}
	return text
}

func formatProject(project *domain.ProjectProfile) string {
	if project == nil {
		return "—"
	}
	if project.RepoURL != "" {
		return fmt.Sprintf("%s (%s)", fallbackText(project.Name), project.RepoURL)
	}
	return fallbackText(project.Name)
}

func formatJobTarget(target *domain.JobTargetRef) string {
	if target == nil {
		return "—"
	}
	if target.CompanyName != "" {
		return fmt.Sprintf("%s (%s)", fallbackText(target.Title), target.CompanyName)
	}
	return fallbackText(target.Title)
}

func formatScore(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatTimeValue(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.UTC().Format(time.RFC3339)
}
