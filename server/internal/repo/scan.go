package repo

import (
	"encoding/json"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

func scanUserProfile(scanner interface{ Scan(dest ...any) error }) (*domain.UserProfile, error) {
	var (
		id                       int64
		targetRole               string
		targetCompanyType        string
		currentStage             string
		applicationDeadline      string
		techStacksJSON           string
		primaryProjectsJSON      string
		selfReportedWeaknessJSON string
		createdAt                string
		updatedAt                string
	)

	if err := scanner.Scan(&id, &targetRole, &targetCompanyType, &currentStage, &applicationDeadline, &techStacksJSON, &primaryProjectsJSON, &selfReportedWeaknessJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	profile := &domain.UserProfile{
		ID:                   id,
		TargetRole:           targetRole,
		TargetCompanyType:    targetCompanyType,
		CurrentStage:         currentStage,
		ApplicationDeadline:  parseNullableTime(applicationDeadline),
		TechStacks:           parseStringList(techStacksJSON),
		PrimaryProjects:      parseStringList(primaryProjectsJSON),
		SelfReportedWeakness: parseStringList(selfReportedWeaknessJSON),
		CreatedAt:            parseTime(createdAt),
		UpdatedAt:            parseTime(updatedAt),
	}

	return profile, nil
}

func scanProjectProfile(scanner interface{ Scan(dest ...any) error }) (*domain.ProjectProfile, error) {
	var (
		id, name, repoURL, defaultBranch, importCommit, summary, techStackJSON, highlightsJSON, challengesJSON string
		tradeoffsJSON, ownershipJSON, followupJSON, importStatus, createdAt, updatedAt                         string
	)

	if err := scanner.Scan(&id, &name, &repoURL, &defaultBranch, &importCommit, &summary, &techStackJSON, &highlightsJSON, &challengesJSON, &tradeoffsJSON, &ownershipJSON, &followupJSON, &importStatus, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	project := &domain.ProjectProfile{
		ID:              id,
		Name:            name,
		RepoURL:         repoURL,
		DefaultBranch:   defaultBranch,
		ImportCommit:    importCommit,
		Summary:         summary,
		TechStack:       parseStringList(techStackJSON),
		Highlights:      parseStringList(highlightsJSON),
		Challenges:      parseStringList(challengesJSON),
		Tradeoffs:       parseStringList(tradeoffsJSON),
		OwnershipPoints: parseStringList(ownershipJSON),
		FollowupPoints:  parseStringList(followupJSON),
		ImportStatus:    importStatus,
		CreatedAt:       parseTime(createdAt),
		UpdatedAt:       parseTime(updatedAt),
	}

	return project, nil
}

func scanProjectImportJob(scanner interface{ Scan(dest ...any) error }) (*domain.ProjectImportJob, error) {
	var id, repoURL, status, stage, message, errorMessage, projectID, projectName, createdAt, updatedAt, startedAt, finishedAt string
	if err := scanner.Scan(&id, &repoURL, &status, &stage, &message, &errorMessage, &projectID, &projectName, &createdAt, &updatedAt, &startedAt, &finishedAt); err != nil {
		return nil, err
	}

	return &domain.ProjectImportJob{
		ID:           id,
		RepoURL:      repoURL,
		Status:       status,
		Stage:        stage,
		Message:      message,
		ErrorMessage: errorMessage,
		ProjectID:    projectID,
		ProjectName:  projectName,
		CreatedAt:    parseTime(createdAt),
		UpdatedAt:    parseTime(updatedAt),
		StartedAt:    parseNullableTime(startedAt),
		FinishedAt:   parseNullableTime(finishedAt),
	}, nil
}

func scanRepoChunk(scanner interface{ Scan(dest ...any) error }) (*domain.RepoChunk, error) {
	var id, projectID, filePath, fileType, content, ftsKey, createdAt string
	var importance float64
	if err := scanner.Scan(&id, &projectID, &filePath, &fileType, &content, &importance, &ftsKey, &createdAt); err != nil {
		return nil, err
	}

	return &domain.RepoChunk{
		ID:         id,
		ProjectID:  projectID,
		FilePath:   filePath,
		FileType:   fileType,
		Content:    content,
		Importance: importance,
		FTSKey:     ftsKey,
		CreatedAt:  parseTime(createdAt),
	}, nil
}

func scanQuestionTemplate(scanner interface{ Scan(dest ...any) error }) (*domain.QuestionTemplate, error) {
	var id, mode, topic, prompt, focusPointsJSON, badAnswersJSON, followupTemplatesJSON, scoreWeightsJSON string
	if err := scanner.Scan(&id, &mode, &topic, &prompt, &focusPointsJSON, &badAnswersJSON, &followupTemplatesJSON, &scoreWeightsJSON); err != nil {
		return nil, err
	}

	weights := map[string]float64{}
	_ = json.Unmarshal([]byte(scoreWeightsJSON), &weights)

	return &domain.QuestionTemplate{
		ID:                id,
		Mode:              mode,
		Topic:             topic,
		Prompt:            prompt,
		FocusPoints:       parseStringList(focusPointsJSON),
		BadAnswers:        parseStringList(badAnswersJSON),
		FollowupTemplates: parseStringList(followupTemplatesJSON),
		ScoreWeights:      weights,
	}, nil
}

func scanTrainingSession(scanner interface{ Scan(dest ...any) error }) (*domain.TrainingSession, error) {
	var id, mode, topic, projectID, intensity, status, startedAt, endedAt, reviewID, createdAt, updatedAt string
	var totalScore float64
	if err := scanner.Scan(&id, &mode, &topic, &projectID, &intensity, &status, &totalScore, &startedAt, &endedAt, &reviewID, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	return &domain.TrainingSession{
		ID:         id,
		Mode:       mode,
		Topic:      topic,
		ProjectID:  projectID,
		Intensity:  intensity,
		Status:     status,
		TotalScore: totalScore,
		StartedAt:  parseNullableTime(startedAt),
		EndedAt:    parseNullableTime(endedAt),
		ReviewID:   reviewID,
		CreatedAt:  parseTime(createdAt),
		UpdatedAt:  parseTime(updatedAt),
	}, nil
}

func scanTrainingTurn(scanner interface{ Scan(dest ...any) error }) (*domain.TrainingTurn, error) {
	var (
		id, sessionID, stage, question, expectedPointsJSON, answer, evaluationJSON, followupQuestion       string
		followupPointsJSON, followupAnswer, followupEvaluationJSON, weaknessHitsJSON, createdAt, updatedAt string
		turnIndex                                                                                          int
	)

	if err := scanner.Scan(&id, &sessionID, &turnIndex, &stage, &question, &expectedPointsJSON, &answer, &evaluationJSON, &followupQuestion, &followupPointsJSON, &followupAnswer, &followupEvaluationJSON, &weaknessHitsJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	turn := &domain.TrainingTurn{
		ID:                    id,
		SessionID:             sessionID,
		TurnIndex:             turnIndex,
		Stage:                 stage,
		Question:              question,
		ExpectedPoints:        parseStringList(expectedPointsJSON),
		Answer:                answer,
		FollowupQuestion:      followupQuestion,
		FollowupExpectedPoint: parseStringList(followupPointsJSON),
		FollowupAnswer:        followupAnswer,
		WeaknessHits:          parseWeaknessHits(weaknessHitsJSON),
		CreatedAt:             parseTime(createdAt),
		UpdatedAt:             parseTime(updatedAt),
	}

	if strings.TrimSpace(evaluationJSON) != "" && evaluationJSON != "null" && evaluationJSON != "{}" {
		evaluation := &domain.EvaluationResult{}
		_ = json.Unmarshal([]byte(evaluationJSON), evaluation)
		turn.Evaluation = evaluation
	}

	if strings.TrimSpace(followupEvaluationJSON) != "" && followupEvaluationJSON != "null" && followupEvaluationJSON != "{}" {
		evaluation := &domain.EvaluationResult{}
		_ = json.Unmarshal([]byte(followupEvaluationJSON), evaluation)
		turn.FollowupEvaluation = evaluation
	}

	return turn, nil
}

func scanReviewCard(scanner interface{ Scan(dest ...any) error }) (*domain.ReviewCard, error) {
	var (
		id, sessionID, overall, topFix, topFixReason, highlightsJSON, gapsJSON string
		suggestedTopicsJSON, nextTrainingFocusJSON, recommendedNextJSON        string
		scoreBreakdownJSON, createdAt                                          string
	)
	if err := scanner.Scan(&id, &sessionID, &overall, &topFix, &topFixReason, &highlightsJSON, &gapsJSON, &suggestedTopicsJSON, &nextTrainingFocusJSON, &recommendedNextJSON, &scoreBreakdownJSON, &createdAt); err != nil {
		return nil, err
	}

	breakdown := map[string]float64{}
	_ = json.Unmarshal([]byte(scoreBreakdownJSON), &breakdown)

	var recommendedNext *domain.NextSession
	if strings.TrimSpace(recommendedNextJSON) != "" && recommendedNextJSON != "null" {
		item := &domain.NextSession{}
		if err := json.Unmarshal([]byte(recommendedNextJSON), item); err == nil {
			recommendedNext = item
		}
	}

	return &domain.ReviewCard{
		ID:                id,
		SessionID:         sessionID,
		Overall:           overall,
		TopFix:            topFix,
		TopFixReason:      topFixReason,
		Highlights:        parseStringList(highlightsJSON),
		Gaps:              parseStringList(gapsJSON),
		SuggestedTopics:   parseStringList(suggestedTopicsJSON),
		NextTrainingFocus: parseStringList(nextTrainingFocusJSON),
		RecommendedNext:   recommendedNext,
		ScoreBreakdown:    breakdown,
		CreatedAt:         parseTime(createdAt),
	}, nil
}

func scanWeaknessTag(scanner interface{ Scan(dest ...any) error }) (*domain.WeaknessTag, error) {
	var id, kind, label, lastSeenAt, evidenceSessionID string
	var severity float64
	var frequency int
	if err := scanner.Scan(&id, &kind, &label, &severity, &frequency, &lastSeenAt, &evidenceSessionID); err != nil {
		return nil, err
	}

	return &domain.WeaknessTag{
		ID:                id,
		Kind:              kind,
		Label:             label,
		Severity:          severity,
		Frequency:         frequency,
		LastSeenAt:        parseTime(lastSeenAt),
		EvidenceSessionID: evidenceSessionID,
	}, nil
}

func parseStringList(raw string) []string {
	items := make([]string, 0)
	_ = json.Unmarshal([]byte(raw), &items)
	return items
}

func parseWeaknessHits(raw string) []domain.WeaknessHit {
	items := make([]domain.WeaknessHit, 0)
	_ = json.Unmarshal([]byte(raw), &items)
	return items
}

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}

	for _, layout := range []string{time.RFC3339Nano, time.DateOnly} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			if layout == time.DateOnly {
				return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			}

			return parsed
		}
	}

	return time.Time{}
}

func parseNullableTime(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	parsed := parseTime(raw)
	if parsed.IsZero() {
		return nil
	}
	return &parsed
}
