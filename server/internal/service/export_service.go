package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

const sessionExportFormatMarkdown = "markdown"

func (s *Service) ExportSession(
	ctx context.Context,
	sessionID string,
	format string,
) (string, []byte, error) {
	if format != sessionExportFormatMarkdown {
		return "", nil, ErrUnsupportedExportFormat
	}

	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return "", nil, err
	}
	if session == nil {
		return "", nil, ErrSessionNotFound
	}

	var review *domain.ReviewCard
	if session.ReviewID != "" {
		review, err = s.repo.GetReview(ctx, session.ReviewID)
		if err != nil {
			return "", nil, err
		}
	}

	filename := fmt.Sprintf("practicehelper-session-%s.md", session.ID)
	content := renderSessionMarkdown(session, review)
	return filename, []byte(content), nil
}

func renderSessionMarkdown(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
) string {
	var builder strings.Builder

	builder.WriteString("# PracticeHelper Session 报告\n\n")
	builder.WriteString("## 基本信息\n\n")
	builder.WriteString(fmt.Sprintf("- Session ID: %s\n", session.ID))
	builder.WriteString(fmt.Sprintf("- 模式: %s\n", session.Mode))
	builder.WriteString(fmt.Sprintf("- 主题: %s\n", fallbackText(session.Topic)))
	builder.WriteString(fmt.Sprintf("- 强度: %s\n", fallbackText(session.Intensity)))
	builder.WriteString(fmt.Sprintf("- 状态: %s\n", session.Status))
	builder.WriteString(fmt.Sprintf("- 最大轮次: %d\n", session.MaxTurns))
	builder.WriteString(fmt.Sprintf("- 总分: %s\n", formatScore(session.TotalScore)))
	builder.WriteString(fmt.Sprintf("- 创建时间: %s\n", formatTimeValue(&session.CreatedAt)))
	builder.WriteString(fmt.Sprintf("- 开始时间: %s\n", formatTimeValue(session.StartedAt)))
	builder.WriteString(fmt.Sprintf("- 结束时间: %s\n", formatTimeValue(session.EndedAt)))
	builder.WriteString(fmt.Sprintf("- 绑定岗位: %s\n", formatJobTarget(session.JobTarget)))
	builder.WriteString(fmt.Sprintf("- 绑定项目: %s\n", formatProject(session.Project)))
	builder.WriteString("\n")

	builder.WriteString("## 训练轮次\n\n")
	if len(session.Turns) == 0 {
		builder.WriteString("> 当前还没有题目或作答记录。\n\n")
	} else {
		for _, turn := range session.Turns {
			builder.WriteString(fmt.Sprintf("### 第 %d 轮\n\n", turn.TurnIndex))
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
				builder.WriteString(fmt.Sprintf("- 得分: %s\n", formatScore(turn.Evaluation.Score)))
				builder.WriteString(fmt.Sprintf("- 总结: %s\n", fallbackText(turn.Evaluation.Headline)))
				builder.WriteString(fmt.Sprintf("- 建议: %s\n", fallbackText(turn.Evaluation.Suggestion)))
				builder.WriteString(fmt.Sprintf("- 追问意图: %s\n", fallbackText(turn.Evaluation.FollowupIntent)))
				builder.WriteString(fmt.Sprintf("- 下一刀追问: %s\n\n", fallbackText(turn.Evaluation.FollowupQuestion)))

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

	builder.WriteString(fmt.Sprintf("- 复盘 ID: %s\n", review.ID))
	builder.WriteString(fmt.Sprintf("- 生成时间: %s\n\n", formatTimeValue(&review.CreatedAt)))

	builder.WriteString("### 总结\n\n")
	builder.WriteString(fallbackText(review.Overall))
	builder.WriteString("\n\n")

	builder.WriteString("### 最该优先修正\n\n")
	builder.WriteString(fmt.Sprintf("- 问题: %s\n", fallbackText(review.TopFix)))
	builder.WriteString(fmt.Sprintf("- 原因: %s\n\n", fallbackText(review.TopFixReason)))

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
		builder.WriteString(fmt.Sprintf("- 模式: %s\n", review.RecommendedNext.Mode))
		builder.WriteString(fmt.Sprintf("- 主题: %s\n", fallbackText(review.RecommendedNext.Topic)))
		builder.WriteString(fmt.Sprintf("- 项目 ID: %s\n", fallbackText(review.RecommendedNext.ProjectID)))
		builder.WriteString(fmt.Sprintf("- 原因: %s\n\n", fallbackText(review.RecommendedNext.Reason)))
	}

	builder.WriteString("### 复盘分项得分\n\n")
	writeScoreBreakdown(&builder, review.ScoreBreakdown, "当前没有复盘分项得分。")

	return builder.String()
}

func writeList(builder *strings.Builder, items []string, empty string) {
	if len(items) == 0 {
		builder.WriteString(fmt.Sprintf("> %s\n\n", empty))
		return
	}

	for _, item := range items {
		builder.WriteString(fmt.Sprintf("- %s\n", fallbackText(item)))
	}
	builder.WriteString("\n")
}

func writeBlock(builder *strings.Builder, content string, empty string) {
	text := strings.TrimSpace(content)
	if text == "" {
		builder.WriteString(fmt.Sprintf("> %s\n\n", empty))
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
		builder.WriteString(fmt.Sprintf("> %s\n\n", empty))
		return
	}

	keys := make([]string, 0, len(breakdown))
	for key := range breakdown {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", key, formatScore(breakdown[key])))
	}
	builder.WriteString("\n")
}

func writeWeaknessHits(
	builder *strings.Builder,
	items []domain.WeaknessHit,
	empty string,
) {
	if len(items) == 0 {
		builder.WriteString(fmt.Sprintf("> %s\n\n", empty))
		return
	}

	for _, item := range items {
		builder.WriteString(
			fmt.Sprintf(
				"- %s / %s / 严重度 %s\n",
				fallbackText(item.Kind),
				fallbackText(item.Label),
				formatScore(item.Severity),
			),
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
