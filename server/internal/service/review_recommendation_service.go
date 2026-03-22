package service

import (
	"context"
	"fmt"
	"strings"

	"practicehelper/server/internal/domain"
)

func (s *Service) normalizeRecommendedNext(
	ctx context.Context,
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	sideEffects *domain.GenerateReviewSideEffects,
) error {
	prerequisiteTopic, err := s.resolveReviewPath(ctx, session, review, sideEffects)
	if err != nil {
		return err
	}
	if prerequisiteTopic == "" {
		return nil
	}
	return s.repo.EnsureKnowledgePrerequisiteEdge(ctx, prerequisiteTopic, normalizeBasicsTopic(session.Topic))
}

func (s *Service) resolveReviewPath(
	ctx context.Context,
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	sideEffects *domain.GenerateReviewSideEffects,
) (string, error) {
	if session == nil || review == nil {
		return "", nil
	}

	suggestion := cloneRecommendedNext(review.RecommendedNext)
	if suggestion == nil && sideEffects != nil && sideEffects.RecommendedNext != nil {
		suggestion = cloneRecommendedNext(sideEffects.RecommendedNext)
	}
	if suggestion == nil {
		suggestion = &domain.NextSession{}
	}

	if session.Mode == domain.ModeProject {
		suggestion.Mode = domain.ModeProject
		if suggestion.ProjectID == "" {
			suggestion.ProjectID = session.ProjectID
		}
		if strings.TrimSpace(suggestion.Reason) == "" {
			suggestion.Reason = buildProjectRecommendedReason(review)
		}
		review.RecommendedNext = suggestion
		return "", nil
	}

	suggestion.Mode = domain.ModeBasics
	suggestion.ProjectID = ""
	suggestion.Topic = resolveRecommendedNextTopic(session, review, suggestion)
	suggestion.Reason = s.buildBasicsRecommendedReason(ctx, session, review, suggestion.Topic)
	review.RecommendedNext = suggestion
	s.normalizeReviewLearningPath(ctx, session, review, suggestion.Topic)
	return resolveReviewPrerequisiteTopic(session, suggestion.Topic), nil
}

func (s *Service) inferReviewPrerequisiteEdge(
	ctx context.Context,
	session *domain.TrainingSession,
	recommendedTopic string,
) error {
	prerequisiteTopic := resolveReviewPrerequisiteTopic(session, recommendedTopic)
	if prerequisiteTopic == "" {
		return nil
	}
	return s.repo.EnsureKnowledgePrerequisiteEdge(ctx, prerequisiteTopic, normalizeBasicsTopic(session.Topic))
}

func resolveReviewPrerequisiteTopic(
	session *domain.TrainingSession,
	recommendedTopic string,
) string {
	if session == nil || session.Mode != domain.ModeBasics {
		return ""
	}

	currentTopic := normalizeBasicsTopic(session.Topic)
	recommendedTopic = normalizeBasicsTopic(recommendedTopic)
	if currentTopic == "" || recommendedTopic == "" {
		return ""
	}
	if currentTopic == domain.BasicsTopicMixed || recommendedTopic == domain.BasicsTopicMixed {
		return ""
	}
	if currentTopic == recommendedTopic {
		return ""
	}

	return recommendedTopic
}

func (s *Service) normalizeReviewLearningPath(
	ctx context.Context,
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	recommendedTopic string,
) {
	if review == nil {
		return
	}

	topic := normalizeBasicsTopic(recommendedTopic)
	if len(review.SuggestedTopics) == 0 && topic != "" && topic != domain.BasicsTopicMixed {
		review.SuggestedTopics = []string{topic}
	}

	if len(review.NextTrainingFocus) > 0 {
		return
	}

	if topic != "" && topic != domain.BasicsTopicMixed {
		projectID := ""
		if session != nil {
			projectID = session.ProjectID
		}
		subgraph, err := s.repo.GetKnowledgeSubgraph(ctx, topic, projectID, 10)
		if err == nil {
			if weakest := pickWeakestKnowledgeNode(subgraph); weakest != nil {
				focus := buildKnowledgeGraphFocus(topic, weakest)
				if focus != "" {
					review.NextTrainingFocus = []string{focus}
					return
				}
			}
		}
	}

	if fallback := buildFallbackLearningFocus(review); fallback != "" {
		review.NextTrainingFocus = []string{fallback}
	}
}

func resolveRecommendedNextTopic(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	suggestion *domain.NextSession,
) string {
	candidates := make([]string, 0, 8)
	if suggestion != nil && suggestion.Topic != "" {
		candidates = append(candidates, suggestion.Topic)
	}
	if review != nil {
		candidates = append(candidates, review.SuggestedTopics...)
		candidates = append(candidates, review.NextTrainingFocus...)
		candidates = append(candidates, review.Gaps...)
		candidates = append(candidates, review.TopFix)
	}
	if session != nil {
		for _, hit := range collectReviewWeaknessHits(session) {
			if hit.Kind == "topic" {
				candidates = append(candidates, hit.Label)
			}
			candidates = append(candidates, matchBasicsTopics(hit.Label)...)
		}
		candidates = append(candidates, session.Topic)
	}

	for _, candidate := range candidates {
		if normalized := normalizeBasicsTopic(candidate); normalized != "" && normalized != domain.BasicsTopicMixed {
			return normalized
		}
		for _, matched := range matchBasicsTopics(candidate) {
			if normalized := normalizeBasicsTopic(matched); normalized != "" && normalized != domain.BasicsTopicMixed {
				return normalized
			}
		}
	}

	return domain.BasicsTopicGo
}

func (s *Service) buildBasicsRecommendedReason(
	ctx context.Context,
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	topic string,
) string {
	focus := firstRecommendedFocus(review)
	if focus == "" {
		focus = topic
	}

	projectID := ""
	if session != nil {
		projectID = session.ProjectID
	}
	subgraph, err := s.repo.GetKnowledgeSubgraph(ctx, topic, projectID, 10)
	if err == nil {
		if weakest := pickWeakestKnowledgeNode(subgraph); weakest != nil {
			if weakest.NodeType == domain.KnowledgeNodeTypeTopic || weakest.Label == topic {
				return fmt.Sprintf(
					"知识图谱里 %s 主题当前掌握度 %.1f/5，先围绕%s继续补。",
					topic,
					weakest.Proficiency,
					focus,
				)
			}
			return fmt.Sprintf(
				"知识图谱里 %s 下的 %s 当前掌握度 %.1f/5，先围绕%s继续补。",
				topic,
				weakest.Label,
				weakest.Proficiency,
				focus,
			)
		}
	}

	return fmt.Sprintf("先围绕%s继续补，这是这轮最影响训练效果的短板。", focus)
}

func buildProjectRecommendedReason(review *domain.ReviewCard) string {
	focus := firstRecommendedFocus(review)
	if focus == "" {
		focus = "当前项目里的关键短板"
	}
	return fmt.Sprintf("先回到当前项目，把%s补扎实，再做下一轮训练。", focus)
}

func firstRecommendedFocus(review *domain.ReviewCard) string {
	if review == nil {
		return ""
	}

	candidates := []string{
		firstNonEmptySliceItem(review.NextTrainingFocus),
		review.TopFix,
		review.TopFixReason,
		firstNonEmptySliceItem(review.Gaps),
		firstNonEmptySliceItem(review.SuggestedTopics),
	}
	for _, item := range candidates {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func buildFallbackLearningFocus(review *domain.ReviewCard) string {
	if review == nil {
		return ""
	}

	candidates := []string{
		review.TopFix,
		firstNonEmptySliceItem(review.Gaps),
		firstNonEmptySliceItem(review.SuggestedTopics),
	}
	for _, item := range candidates {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func firstNonEmptySliceItem(items []string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func pickWeakestKnowledgeNode(subgraph *domain.KnowledgeSubgraph) *domain.KnowledgeNode {
	if subgraph == nil || len(subgraph.Nodes) == 0 {
		return nil
	}

	var weakest *domain.KnowledgeNode
	for i := range subgraph.Nodes {
		node := &subgraph.Nodes[i]
		if weakest == nil {
			weakest = node
			continue
		}
		if node.NodeType != domain.KnowledgeNodeTypeTopic && weakest.NodeType == domain.KnowledgeNodeTypeTopic {
			weakest = node
			continue
		}
		if node.Proficiency < weakest.Proficiency {
			weakest = node
			continue
		}
		if node.Proficiency == weakest.Proficiency && node.Confidence > weakest.Confidence {
			weakest = node
		}
	}
	return weakest
}

func buildKnowledgeGraphFocus(topic string, node *domain.KnowledgeNode) string {
	if node == nil {
		return ""
	}

	label := displayKnowledgeNodeLabel(node.Label)
	if label == "" {
		return ""
	}
	if node.NodeType == domain.KnowledgeNodeTypeTopic || node.Label == topic {
		return fmt.Sprintf("围绕 %s 主题短板继续补", topic)
	}
	return fmt.Sprintf("围绕 %s 下的 %s 继续补", topic, label)
}

func displayKnowledgeNodeLabel(label string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(label, "_", " "))
	normalized = strings.ReplaceAll(normalized, "-", " ")
	return normalized
}

func cloneRecommendedNext(next *domain.NextSession) *domain.NextSession {
	if next == nil {
		return nil
	}
	copy := *next
	return &copy
}
