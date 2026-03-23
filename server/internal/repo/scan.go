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
		activeJobTargetID        string
		createdAt                string
		updatedAt                string
	)

	if err := scanner.Scan(&id, &targetRole, &targetCompanyType, &currentStage, &applicationDeadline, &techStacksJSON, &primaryProjectsJSON, &selfReportedWeaknessJSON, &activeJobTargetID, &createdAt, &updatedAt); err != nil {
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
		ActiveJobTargetID:    activeJobTargetID,
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

func scanJobTarget(scanner interface{ Scan(dest ...any) error }) (*domain.JobTarget, error) {
	var (
		id, title, companyName, sourceText, latestAnalysisID, latestAnalysisStatus string
		lastUsedAt, createdAt, updatedAt                                           string
	)

	if err := scanner.Scan(
		&id,
		&title,
		&companyName,
		&sourceText,
		&latestAnalysisID,
		&latestAnalysisStatus,
		&lastUsedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	return &domain.JobTarget{
		ID:                   id,
		Title:                title,
		CompanyName:          companyName,
		SourceText:           sourceText,
		LatestAnalysisID:     latestAnalysisID,
		LatestAnalysisStatus: latestAnalysisStatus,
		LastUsedAt:           parseNullableTime(lastUsedAt),
		CreatedAt:            parseTime(createdAt),
		UpdatedAt:            parseTime(updatedAt),
	}, nil
}

func scanJobTargetAnalysisRun(scanner interface{ Scan(dest ...any) error }) (*domain.JobTargetAnalysisRun, error) {
	var (
		id, jobTargetID, sourceTextSnapshot, status, errorMessage, summary string
		mustHaveSkillsJSON, bonusSkillsJSON, responsibilitiesJSON          string
		evaluationFocusJSON, createdAt, finishedAt                         string
	)

	if err := scanner.Scan(
		&id,
		&jobTargetID,
		&sourceTextSnapshot,
		&status,
		&errorMessage,
		&summary,
		&mustHaveSkillsJSON,
		&bonusSkillsJSON,
		&responsibilitiesJSON,
		&evaluationFocusJSON,
		&createdAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}

	return &domain.JobTargetAnalysisRun{
		ID:                 id,
		JobTargetID:        jobTargetID,
		SourceTextSnapshot: sourceTextSnapshot,
		Status:             status,
		ErrorMessage:       errorMessage,
		Summary:            summary,
		MustHaveSkills:     parseStringList(mustHaveSkillsJSON),
		BonusSkills:        parseStringList(bonusSkillsJSON),
		Responsibilities:   parseStringList(responsibilitiesJSON),
		EvaluationFocus:    parseStringList(evaluationFocusJSON),
		CreatedAt:          parseTime(createdAt),
		FinishedAt:         parseNullableTime(finishedAt),
	}, nil
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

func scanRepoChunkEmbeddingRecord(scanner interface{ Scan(dest ...any) error }) (*domain.RepoChunkEmbeddingRecord, error) {
	var (
		id, repoChunkID, projectID, contentHash string
		modelName, vectorStoreID, status        string
		lastError, lastIndexedAt                string
		createdAt, updatedAt                    string
	)
	var vectorDim int
	if err := scanner.Scan(
		&id,
		&repoChunkID,
		&projectID,
		&contentHash,
		&modelName,
		&vectorStoreID,
		&vectorDim,
		&status,
		&lastError,
		&lastIndexedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	return &domain.RepoChunkEmbeddingRecord{
		ID:            id,
		RepoChunkID:   repoChunkID,
		ProjectID:     projectID,
		ContentHash:   contentHash,
		ModelName:     modelName,
		VectorStoreID: vectorStoreID,
		VectorDim:     vectorDim,
		Status:        status,
		LastError:     lastError,
		LastIndexedAt: parseNullableTime(lastIndexedAt),
		CreatedAt:     parseTime(createdAt),
		UpdatedAt:     parseTime(updatedAt),
	}, nil
}

func scanRepoChunkEmbeddingJob(scanner interface{ Scan(dest ...any) error }) (*domain.RepoChunkEmbeddingJob, error) {
	var (
		id, repoChunkID, projectID, status string
		errorMessage, claimToken           string
		claimExpiresAt, createdAt          string
		updatedAt, startedAt, finishedAt   string
	)
	var attemptCount int
	if err := scanner.Scan(
		&id,
		&repoChunkID,
		&projectID,
		&status,
		&attemptCount,
		&errorMessage,
		&claimToken,
		&claimExpiresAt,
		&createdAt,
		&updatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}

	return &domain.RepoChunkEmbeddingJob{
		ID:             id,
		RepoChunkID:    repoChunkID,
		ProjectID:      projectID,
		Status:         status,
		AttemptCount:   attemptCount,
		ErrorMessage:   errorMessage,
		ClaimToken:     claimToken,
		ClaimExpiresAt: parseNullableTime(claimExpiresAt),
		CreatedAt:      parseTime(createdAt),
		UpdatedAt:      parseTime(updatedAt),
		StartedAt:      parseNullableTime(startedAt),
		FinishedAt:     parseNullableTime(finishedAt),
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
	var (
		id, mode, topic, projectID, jobTargetID, jobTargetAnalysisID string
		promptSetID, promptSetLabel, promptSetStatus                 string
		promptOverlayJSON, promptOverlayHash                         string
		intensity, status, startedAt, endedAt, reviewID              string
		createdAt, updatedAt                                         string
	)
	var totalScore float64
	var maxTurns int
	if err := scanner.Scan(
		&id,
		&mode,
		&topic,
		&projectID,
		&jobTargetID,
		&jobTargetAnalysisID,
		&promptSetID,
		&promptSetLabel,
		&promptSetStatus,
		&promptOverlayJSON,
		&promptOverlayHash,
		&intensity,
		&status,
		&maxTurns,
		&totalScore,
		&startedAt,
		&endedAt,
		&reviewID,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	return &domain.TrainingSession{
		ID:                  id,
		Mode:                mode,
		Topic:               topic,
		ProjectID:           projectID,
		JobTargetID:         jobTargetID,
		JobTargetAnalysisID: jobTargetAnalysisID,
		PromptSetID:         promptSetID,
		PromptOverlay:       parsePromptOverlayJSON(promptOverlayJSON),
		PromptOverlayHash:   promptOverlayHash,
		Intensity:           intensity,
		Status:              status,
		MaxTurns:            maxTurns,
		TotalScore:          totalScore,
		StartedAt:           parseNullableTime(startedAt),
		EndedAt:             parseNullableTime(endedAt),
		ReviewID:            reviewID,
		CreatedAt:           parseTime(createdAt),
		UpdatedAt:           parseTime(updatedAt),
		PromptSet:           parsePromptSetSummary(promptSetID, promptSetLabel, promptSetStatus),
	}, nil
}

func scanTrainingTurn(scanner interface{ Scan(dest ...any) error }) (*domain.TrainingTurn, error) {
	var (
		id, sessionID, stage, question, expectedPointsJSON string
		answer, evaluationJSON, weaknessHitsJSON           string
		createdAt, updatedAt                               string
		turnIndex                                          int
	)

	if err := scanner.Scan(&id, &sessionID, &turnIndex, &stage, &question, &expectedPointsJSON, &answer, &evaluationJSON, &weaknessHitsJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	turn := &domain.TrainingTurn{
		ID:             id,
		SessionID:      sessionID,
		TurnIndex:      turnIndex,
		Stage:          stage,
		Question:       question,
		ExpectedPoints: parseStringList(expectedPointsJSON),
		Answer:         answer,
		WeaknessHits:   parseWeaknessHits(weaknessHitsJSON),
		CreatedAt:      parseTime(createdAt),
		UpdatedAt:      parseTime(updatedAt),
	}

	if strings.TrimSpace(evaluationJSON) != "" && evaluationJSON != "null" && evaluationJSON != "{}" {
		evaluation := &domain.EvaluationResult{}
		_ = json.Unmarshal([]byte(evaluationJSON), evaluation)
		turn.Evaluation = evaluation
	}

	return turn, nil
}

func scanReviewCard(scanner interface{ Scan(dest ...any) error }) (*domain.ReviewCard, error) {
	var (
		id, sessionID, jobTargetID, jobTargetAnalysisID, overall, topFix, topFixReason string
		promptSetID, promptSetLabel, promptSetStatus                                   string
		promptOverlayJSON, promptOverlayHash                                           string
		highlightsJSON, gapsJSON, suggestedTopicsJSON, nextTrainingFocusJSON           string
		recommendedNextJSON, retrievalTraceJSON, scoreBreakdownJSON, createdAt         string
	)
	if err := scanner.Scan(
		&id,
		&sessionID,
		&jobTargetID,
		&jobTargetAnalysisID,
		&promptSetID,
		&promptSetLabel,
		&promptSetStatus,
		&promptOverlayJSON,
		&promptOverlayHash,
		&overall,
		&topFix,
		&topFixReason,
		&highlightsJSON,
		&gapsJSON,
		&suggestedTopicsJSON,
		&nextTrainingFocusJSON,
		&recommendedNextJSON,
		&retrievalTraceJSON,
		&scoreBreakdownJSON,
		&createdAt,
	); err != nil {
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

	var retrievalTrace *domain.RetrievalTrace
	if strings.TrimSpace(retrievalTraceJSON) != "" && retrievalTraceJSON != "null" {
		item := &domain.RetrievalTrace{}
		if err := json.Unmarshal([]byte(retrievalTraceJSON), item); err == nil {
			retrievalTrace = item
		}
	}

	return &domain.ReviewCard{
		ID:                  id,
		SessionID:           sessionID,
		JobTargetID:         jobTargetID,
		JobTargetAnalysisID: jobTargetAnalysisID,
		PromptSetID:         promptSetID,
		PromptOverlay:       parsePromptOverlayJSON(promptOverlayJSON),
		PromptOverlayHash:   promptOverlayHash,
		Overall:             overall,
		TopFix:              topFix,
		TopFixReason:        topFixReason,
		Highlights:          parseStringList(highlightsJSON),
		Gaps:                parseStringList(gapsJSON),
		SuggestedTopics:     parseStringList(suggestedTopicsJSON),
		NextTrainingFocus:   parseStringList(nextTrainingFocusJSON),
		RecommendedNext:     recommendedNext,
		RetrievalTrace:      retrievalTrace,
		ScoreBreakdown:      breakdown,
		CreatedAt:           parseTime(createdAt),
		PromptSet:           parsePromptSetSummary(promptSetID, promptSetLabel, promptSetStatus),
	}, nil
}

func parsePromptSetSummary(id, label, status string) *domain.PromptSetSummary {
	if strings.TrimSpace(id) == "" {
		return nil
	}

	return &domain.PromptSetSummary{
		ID:     id,
		Label:  label,
		Status: status,
	}
}

func parsePromptOverlayJSON(raw string) *domain.PromptOverlay {
	if strings.TrimSpace(raw) == "" || raw == "null" {
		return nil
	}

	var overlay domain.PromptOverlay
	if err := json.Unmarshal([]byte(raw), &overlay); err != nil {
		return nil
	}
	if overlay.Tone == "" &&
		overlay.DetailLevel == "" &&
		overlay.FollowupIntensity == "" &&
		overlay.AnswerLanguage == "" &&
		len(overlay.FocusTags) == 0 &&
		overlay.CustomInstruction == "" {
		return nil
	}
	return &overlay
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

func scanKnowledgeNode(scanner interface{ Scan(dest ...any) error }) (*domain.KnowledgeNode, error) {
	var (
		id, scopeType, scopeID, parentID, label, nodeType string
		lastAssessedAt, createdAt, updatedAt              string
	)
	var proficiency, confidence float64
	var hitCount int
	if err := scanner.Scan(
		&id,
		&scopeType,
		&scopeID,
		&parentID,
		&label,
		&nodeType,
		&proficiency,
		&confidence,
		&hitCount,
		&lastAssessedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	return &domain.KnowledgeNode{
		ID:             id,
		ScopeType:      scopeType,
		ScopeID:        scopeID,
		ParentID:       parentID,
		Label:          label,
		NodeType:       nodeType,
		Proficiency:    proficiency,
		Confidence:     confidence,
		HitCount:       hitCount,
		LastAssessedAt: parseNullableTime(lastAssessedAt),
		CreatedAt:      parseTime(createdAt),
		UpdatedAt:      parseTime(updatedAt),
	}, nil
}

func scanKnowledgeEdge(scanner interface{ Scan(dest ...any) error }) (*domain.KnowledgeEdge, error) {
	var sourceID, targetID, edgeType, createdAt string
	if err := scanner.Scan(&sourceID, &targetID, &edgeType, &createdAt); err != nil {
		return nil, err
	}

	return &domain.KnowledgeEdge{
		SourceID:  sourceID,
		TargetID:  targetID,
		EdgeType:  edgeType,
		CreatedAt: parseTime(createdAt),
	}, nil
}

func scanAgentObservation(scanner interface{ Scan(dest ...any) error }) (*domain.AgentObservation, error) {
	var (
		id, sessionID, scopeType, scopeID, topic string
		category, content, tagsJSON              string
		createdAt, archivedAt                    string
	)
	var relevance float64
	if err := scanner.Scan(
		&id,
		&sessionID,
		&scopeType,
		&scopeID,
		&topic,
		&category,
		&content,
		&tagsJSON,
		&relevance,
		&createdAt,
		&archivedAt,
	); err != nil {
		return nil, err
	}

	return &domain.AgentObservation{
		ID:         id,
		SessionID:  sessionID,
		ScopeType:  scopeType,
		ScopeID:    scopeID,
		Topic:      topic,
		Category:   category,
		Content:    content,
		Tags:       parseStringList(tagsJSON),
		Relevance:  relevance,
		CreatedAt:  parseTime(createdAt),
		ArchivedAt: parseNullableTime(archivedAt),
	}, nil
}

func scanSessionMemorySummary(scanner interface{ Scan(dest ...any) error }) (*domain.SessionMemorySummary, error) {
	var (
		id, sessionID, mode, topic, projectID, jobTargetID, promptSetID string
		summary, strengthsJSON, gapsJSON                                string
		misconceptionsJSON, growthJSON, recommendedFocusJSON            string
		createdAt, updatedAt                                            string
	)
	var salience float64
	if err := scanner.Scan(
		&id,
		&sessionID,
		&mode,
		&topic,
		&projectID,
		&jobTargetID,
		&promptSetID,
		&summary,
		&strengthsJSON,
		&gapsJSON,
		&misconceptionsJSON,
		&growthJSON,
		&recommendedFocusJSON,
		&salience,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	return &domain.SessionMemorySummary{
		ID:               id,
		SessionID:        sessionID,
		Mode:             mode,
		Topic:            topic,
		ProjectID:        projectID,
		JobTargetID:      jobTargetID,
		PromptSetID:      promptSetID,
		Summary:          summary,
		Strengths:        parseStringList(strengthsJSON),
		Gaps:             parseStringList(gapsJSON),
		Misconceptions:   parseStringList(misconceptionsJSON),
		GrowthSignals:    parseStringList(growthJSON),
		RecommendedFocus: parseStringList(recommendedFocusJSON),
		Salience:         salience,
		CreatedAt:        parseTime(createdAt),
		UpdatedAt:        parseTime(updatedAt),
	}, nil
}

func scanMemoryIndexEntry(scanner interface{ Scan(dest ...any) error }) (*domain.MemoryIndexEntry, error) {
	var (
		id, memoryType, scopeType, scopeID, topic, projectID, sessionID, jobTargetID string
		tagsJSON, entitiesJSON, summary, refTable, refID                             string
		createdAt, updatedAt                                                         string
	)
	var salience, confidence, freshness float64
	if err := scanner.Scan(
		&id,
		&memoryType,
		&scopeType,
		&scopeID,
		&topic,
		&projectID,
		&sessionID,
		&jobTargetID,
		&tagsJSON,
		&entitiesJSON,
		&summary,
		&salience,
		&confidence,
		&freshness,
		&refTable,
		&refID,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	return &domain.MemoryIndexEntry{
		ID:          id,
		MemoryType:  memoryType,
		ScopeType:   scopeType,
		ScopeID:     scopeID,
		Topic:       topic,
		ProjectID:   projectID,
		SessionID:   sessionID,
		JobTargetID: jobTargetID,
		Tags:        parseStringList(tagsJSON),
		Entities:    parseStringList(entitiesJSON),
		Summary:     summary,
		Salience:    salience,
		Confidence:  confidence,
		Freshness:   freshness,
		RefTable:    refTable,
		RefID:       refID,
		CreatedAt:   parseTime(createdAt),
		UpdatedAt:   parseTime(updatedAt),
	}, nil
}

func scanMemoryEmbeddingRecord(
	scanner interface{ Scan(dest ...any) error },
) (*domain.MemoryEmbeddingRecord, error) {
	var (
		id, memoryIndexID, memoryType, refTable, refID string
		contentHash, modelName, vectorStoreID, status  string
		lastError, lastIndexedAt, createdAt, updatedAt string
		vectorDim                                      int
	)
	if err := scanner.Scan(
		&id,
		&memoryIndexID,
		&memoryType,
		&refTable,
		&refID,
		&contentHash,
		&modelName,
		&vectorStoreID,
		&vectorDim,
		&status,
		&lastError,
		&lastIndexedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	return &domain.MemoryEmbeddingRecord{
		ID:            id,
		MemoryIndexID: memoryIndexID,
		MemoryType:    memoryType,
		RefTable:      refTable,
		RefID:         refID,
		ContentHash:   contentHash,
		ModelName:     modelName,
		VectorStoreID: vectorStoreID,
		VectorDim:     vectorDim,
		Status:        status,
		LastError:     lastError,
		LastIndexedAt: parseNullableTime(lastIndexedAt),
		CreatedAt:     parseTime(createdAt),
		UpdatedAt:     parseTime(updatedAt),
	}, nil
}

func scanMemoryEmbeddingJob(
	scanner interface{ Scan(dest ...any) error },
) (*domain.MemoryEmbeddingJob, error) {
	var (
		id, memoryIndexID, memoryType, refTable, refID string
		status, errorMessage, claimToken               string
		claimExpiresAt, createdAt, updatedAt           string
		startedAt, finishedAt                          string
		attemptCount                                   int
	)
	if err := scanner.Scan(
		&id,
		&memoryIndexID,
		&memoryType,
		&refTable,
		&refID,
		&status,
		&attemptCount,
		&errorMessage,
		&claimToken,
		&claimExpiresAt,
		&createdAt,
		&updatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}

	return &domain.MemoryEmbeddingJob{
		ID:             id,
		MemoryIndexID:  memoryIndexID,
		MemoryType:     memoryType,
		RefTable:       refTable,
		RefID:          refID,
		Status:         status,
		AttemptCount:   attemptCount,
		ErrorMessage:   errorMessage,
		ClaimToken:     claimToken,
		ClaimExpiresAt: parseNullableTime(claimExpiresAt),
		CreatedAt:      parseTime(createdAt),
		UpdatedAt:      parseTime(updatedAt),
		StartedAt:      parseNullableTime(startedAt),
		FinishedAt:     parseNullableTime(finishedAt),
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
