package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

func (s *Service) ListWeaknesses(ctx context.Context) ([]domain.WeaknessTag, error) {
	return s.repo.ListWeaknesses(ctx, 20)
}

func (s *Service) lookupSessionContext(ctx context.Context, session *domain.TrainingSession) ([]domain.RepoChunk, error) {
	if session.Mode != domain.ModeProject || session.Project == nil {
		return nil, nil
	}

	query := strings.Join(append(session.Project.FollowupPoints, session.Project.Summary), " ")
	return s.repo.SearchProjectChunks(ctx, session.Project.ID, query, 6)
}

func (s *Service) coolDownSessionWeakness(
	ctx context.Context,
	session *domain.TrainingSession,
	questionText string,
	hits []domain.WeaknessHit,
) error {
	// 这里是“答得好就给弱点降温”的启发式修正，不追求精确评分模型。
	// 命中的具体弱点和本轮主维度（topic/project）都会被轻微衰减；
	// 即便降温失败，也不能反向影响训练主流程，所以这里统一忽略 repo 写入错误。
	for _, hit := range hits {
		_ = s.repo.RelieveWeakness(ctx, hit.Kind, hit.Label, 0.18)
	}

	// 如果当前题目文本已经明确对应某个历史弱项 label，说明这次高分回答对那个点有直接修复价值。
	// 这里额外做一次按题面文本匹配的降温，避免“答对了同一题型，但旧标签完全不降”的僵硬体感。
	_ = s.repo.RelieveWeaknessesMatchingText(
		ctx,
		[]string{"topic", "depth", "detail", "followup_breakdown"},
		questionText,
		0.28,
	)

	if session.Mode == domain.ModeBasics && session.Topic != "" {
		_ = s.repo.RelieveWeakness(ctx, "topic", session.Topic, 0.15)
	}
	if session.Mode == domain.ModeProject && session.Project != nil {
		_ = s.repo.RelieveWeakness(ctx, "project", session.Project.Name, 0.15)
	}

	return nil
}

func buildTodayFocus(profile *domain.UserProfile, weaknesses []domain.WeaknessTag) string {
	if len(weaknesses) > 0 {
		return fmt.Sprintf("今天优先补 %s：%s", formatWeaknessKindLabel(weaknesses[0].Kind), weaknesses[0].Label)
	}

	if profile != nil && len(profile.PrimaryProjects) > 0 {
		return fmt.Sprintf("今天先做一轮项目表达训练，主讲 %s", profile.PrimaryProjects[0])
	}

	return "先完成画像初始化，然后做一轮 Redis 或 Go 基础训练。"
}

func buildRecommendedTrack(profile *domain.UserProfile, weaknesses []domain.WeaknessTag) string {
	if len(weaknesses) > 0 {
		switch weaknesses[0].Kind {
		case "topic":
			return fmt.Sprintf("%s 专项训练", strings.ToUpper(weaknesses[0].Label))
		case "project":
			return fmt.Sprintf("%s 项目专项", weaknesses[0].Label)
		case "expression":
			return fmt.Sprintf("围绕「%s」做表达方式专项", weaknesses[0].Label)
		case "followup_breakdown":
			return fmt.Sprintf("围绕「%s」做追问抗压专项", weaknesses[0].Label)
		case "depth":
			return fmt.Sprintf("围绕「%s」做展开深挖专项", weaknesses[0].Label)
		case "detail":
			return fmt.Sprintf("围绕「%s」做细节补强专项", weaknesses[0].Label)
		default:
			return fmt.Sprintf("围绕「%s」做针对性训练", weaknesses[0].Label)
		}
	}

	if profile != nil && profile.TargetRole != "" {
		return fmt.Sprintf("%s 目标岗位基础训练", profile.TargetRole)
	}

	return "Go 后端 + Agent 方向基础训练"
}

func mergeWeaknessHits(base []domain.WeaknessHit, extra []domain.WeaknessHit) []domain.WeaknessHit {
	result := append([]domain.WeaknessHit{}, base...)
	result = append(result, extra...)
	return result
}

func formatWeaknessKindLabel(kind string) string {
	switch kind {
	case "topic":
		return "知识点"
	case "project":
		return "项目表达"
	case "expression":
		return "表达方式"
	case "followup_breakdown":
		return "追问应对"
	case "depth":
		return "展开深度"
	case "detail":
		return "细节支撑"
	default:
		return kind
	}
}

func daysUntilDeadline(deadline time.Time) int {
	today := time.Now().UTC()
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	target := deadline.UTC()
	targetStart := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, time.UTC)

	return int(targetStart.Sub(todayStart).Hours() / 24)
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
