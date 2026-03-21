package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/sidecar"
)

func TestSaveProfileSeedsKnowledgeFromTechStacks(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	svc := New(store, nil)
	_, err = svc.SaveProfile(context.Background(), domain.UserProfileInput{
		TargetRole:           "Backend Engineer",
		TargetCompanyType:    "Startup",
		CurrentStage:         "Interview",
		TechStacks:           []string{"Rust"},
		PrimaryProjects:      []string{"PracticeHelper"},
		SelfReportedWeakness: []string{"系统设计"},
	})
	if err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}

	subgraph, err := store.GetKnowledgeSubgraph(context.Background(), "rust", "", 4)
	if err != nil {
		t.Fatalf("GetKnowledgeSubgraph() error = %v", err)
	}
	if len(subgraph.Nodes) == 0 {
		t.Fatal("expected rust to be seeded into knowledge graph")
	}
	if subgraph.Nodes[0].Label != "rust" {
		t.Fatalf("expected seeded node label rust, got %q", subgraph.Nodes[0].Label)
	}
}

func TestCreateSessionPreloadsAgentContext(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	sidecarServer := newAgentContextTestServer(t)
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	_, err = svc.SaveProfile(context.Background(), domain.UserProfileInput{
		TargetRole:           "Backend Engineer",
		TargetCompanyType:    "Startup",
		CurrentStage:         "Interview",
		TechStacks:           []string{"Go", "Redis"},
		PrimaryProjects:      []string{"PracticeHelper"},
		SelfReportedWeakness: []string{"表达"},
	})
	if err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}

	session, err := svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Intensity: "standard",
		MaxTurns:  2,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session == nil || session.ID == "" {
		t.Fatal("expected created session")
	}
}

func TestPersistReviewStoresSessionMemorySummary(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:          "sess_memory_summary",
		Mode:        domain.ModeBasics,
		Topic:       domain.BasicsTopicRedis,
		PromptSetID: "stable-v1",
		Intensity:   "standard",
		Status:      domain.StatusReviewPending,
		MaxTurns:    2,
		TotalScore:  82,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_memory_summary",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
		WeaknessHits: []domain.WeaknessHit{{
			Kind:     "detail",
			Label:    "缓存击穿处理",
			Severity: 0.7,
		}},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	review := &domain.ReviewCard{
		Overall:           "主线已经清楚，但故障处理细节还不够扎实。",
		Highlights:        []string{"能说清 Redis 的性能来源"},
		Gaps:              []string{"缓存击穿处理步骤不完整"},
		SuggestedTopics:   []string{"redis"},
		NextTrainingFocus: []string{"缓存击穿止血顺序"},
	}
	if _, err := svc.persistReview(
		context.Background(),
		session,
		review,
		&domain.GenerateReviewSideEffects{},
	); err != nil {
		t.Fatalf("persistReview() error = %v", err)
	}

	summary, err := store.GetSessionMemorySummary(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("GetSessionMemorySummary() error = %v", err)
	}
	if summary == nil {
		t.Fatal("expected session memory summary to be persisted")
	}
	if summary.Summary != review.Overall {
		t.Fatalf("expected summary overall to be saved, got %q", summary.Summary)
	}
	if len(summary.RecommendedFocus) == 0 || summary.RecommendedFocus[0] != "缓存击穿止血顺序" {
		t.Fatalf("expected recommended focus to be saved, got %#v", summary.RecommendedFocus)
	}
}

func TestGetAgentContextUsesMemoryIndexPlanner(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.CreateObservations(ctx, "sess_obs_project", []domain.AgentObservation{
		{
			ID:        "obs_project",
			SessionID: "sess_obs_project",
			ScopeType: domain.MemoryScopeProject,
			ScopeID:   "proj_1",
			Topic:     "redis",
			Category:  "pattern",
			Content:   "项目里的缓存一致性答法更值得优先调取。",
			Relevance: 0.55,
		},
		{
			ID:        "obs_topic",
			SessionID: "sess_obs_topic",
			ScopeType: domain.MemoryScopeGlobal,
			Topic:     "redis",
			Category:  "pattern",
			Content:   "通用 Redis 观察作为第二优先级。",
			Relevance: 0.95,
		},
		{
			ID:        "obs_noise",
			SessionID: "sess_obs_noise",
			ScopeType: domain.MemoryScopeGlobal,
			Topic:     "kafka",
			Category:  "pattern",
			Content:   "不相关的 Kafka 观察不应被带进来。",
			Relevance: 0.99,
		},
	}); err != nil {
		t.Fatalf("CreateObservations() error = %v", err)
	}

	if err := store.UpsertSessionMemorySummary(ctx, &domain.SessionMemorySummary{
		ID:               "sm_project",
		SessionID:        "sess_prev_project",
		Mode:             domain.ModeProject,
		Topic:            "redis",
		ProjectID:        "proj_1",
		Summary:          "项目场景下的 Redis 总结应该排第一。",
		RecommendedFocus: []string{"项目里的缓存一致性取舍"},
		Salience:         0.6,
	}); err != nil {
		t.Fatalf("UpsertSessionMemorySummary() project error = %v", err)
	}
	if err := store.UpsertSessionMemorySummary(ctx, &domain.SessionMemorySummary{
		ID:               "sm_topic",
		SessionID:        "sess_prev_topic",
		Mode:             domain.ModeBasics,
		Topic:            "redis",
		Summary:          "同 topic 的历史总结应该排在第二层。",
		RecommendedFocus: []string{"Redis 基础表达"},
		Salience:         0.95,
	}); err != nil {
		t.Fatalf("UpsertSessionMemorySummary() topic error = %v", err)
	}
	if err := store.UpsertSessionMemorySummary(ctx, &domain.SessionMemorySummary{
		ID:               "sm_noise",
		SessionID:        "sess_prev_noise",
		Mode:             domain.ModeBasics,
		Topic:            "kafka",
		Summary:          "不相关的 Kafka 总结不应被带进来。",
		RecommendedFocus: []string{"Kafka 基础"},
		Salience:         0.99,
	}); err != nil {
		t.Fatalf("UpsertSessionMemorySummary() noise error = %v", err)
	}

	svc := New(store, nil)
	agentContext, err := svc.getAgentContext(ctx, agentContextParams{
		Topic:               "redis",
		ProjectID:           "proj_1",
		SessionID:           "sess_current",
		ObservationLimit:    2,
		SessionSummaryLimit: 2,
		KnowledgeNodeLimit:  2,
	})
	if err != nil {
		t.Fatalf("getAgentContext() error = %v", err)
	}

	if len(agentContext.Observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(agentContext.Observations))
	}
	if agentContext.Observations[0].ID != "obs_project" {
		t.Fatalf("expected project observation first, got %q", agentContext.Observations[0].ID)
	}
	if agentContext.Observations[1].ID != "obs_topic" {
		t.Fatalf("expected topic observation second, got %q", agentContext.Observations[1].ID)
	}

	if len(agentContext.SessionSummaries) != 2 {
		t.Fatalf("expected 2 session summaries, got %d", len(agentContext.SessionSummaries))
	}
	if agentContext.SessionSummaries[0].ID != "sm_project" {
		t.Fatalf("expected project summary first, got %q", agentContext.SessionSummaries[0].ID)
	}
	if agentContext.SessionSummaries[1].ID != "sm_topic" {
		t.Fatalf("expected topic summary second, got %q", agentContext.SessionSummaries[1].ID)
	}
}

func TestPersistReviewBuildsKnowledgeGraphBackedRecommendedNext(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.EnsureKnowledgeSeeds(ctx); err != nil {
		t.Fatalf("EnsureKnowledgeSeeds() error = %v", err)
	}

	seedSubgraph, err := store.GetKnowledgeSubgraph(ctx, domain.BasicsTopicRedis, "", 2)
	if err != nil {
		t.Fatalf("GetKnowledgeSubgraph() seed error = %v", err)
	}
	if len(seedSubgraph.Nodes) == 0 {
		t.Fatal("expected redis topic seed node")
	}
	root := seedSubgraph.Nodes[0]
	if err := store.UpsertKnowledgeNodes(ctx, "sess_graph_reason_seed", []domain.KnowledgeUpdate{
		{
			ScopeType:   domain.MemoryScopeGlobal,
			ParentID:    root.ID,
			Label:       "cache_consistency",
			NodeType:    domain.KnowledgeNodeTypeConcept,
			Proficiency: 1.5,
			Confidence:  0.9,
			Evidence:    "回答里能说目标，但 trade-off 还不够完整。",
		},
	}); err != nil {
		t.Fatalf("UpsertKnowledgeNodes() error = %v", err)
	}

	session := &domain.TrainingSession{
		ID:          "sess_review_recommendation",
		Mode:        domain.ModeBasics,
		Topic:       domain.BasicsTopicRedis,
		PromptSetID: "stable-v1",
		Intensity:   "standard",
		Status:      domain.StatusReviewPending,
		MaxTurns:    2,
		TotalScore:  74,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_review_recommendation",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 如何处理缓存一致性？",
		ExpectedPoints: []string{"一致性目标", "失效策略"},
		WeaknessHits: []domain.WeaknessHit{{
			Kind:     "detail",
			Label:    "缓存一致性",
			Severity: 0.82,
		}},
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	review := &domain.ReviewCard{
		Overall:           "主线能站住，但缓存一致性 trade-off 还是偏虚。",
		TopFix:            "补缓存一致性取舍",
		TopFixReason:      "这是当前最影响说服力的短板。",
		Highlights:        []string{"主线清楚"},
		Gaps:              []string{"缓存一致性 trade-off 不够具体"},
		SuggestedTopics:   []string{domain.BasicsTopicRedis},
		NextTrainingFocus: []string{"缓存一致性取舍"},
	}
	if _, err := svc.persistReview(ctx, session, review, &domain.GenerateReviewSideEffects{}); err != nil {
		t.Fatalf("persistReview() error = %v", err)
	}

	savedReview, err := store.GetReview(ctx, session.ReviewID)
	if err != nil {
		t.Fatalf("GetReview() error = %v", err)
	}
	if savedReview == nil || savedReview.RecommendedNext == nil {
		t.Fatalf("expected recommended next to be persisted, got %#v", savedReview)
	}
	if savedReview.RecommendedNext.Mode != domain.ModeBasics {
		t.Fatalf("expected basics mode, got %#v", savedReview.RecommendedNext)
	}
	if savedReview.RecommendedNext.Topic != domain.BasicsTopicRedis {
		t.Fatalf("expected redis topic, got %#v", savedReview.RecommendedNext)
	}
	if savedReview.RecommendedNext.Reason == "" {
		t.Fatal("expected recommended next reason to be generated")
	}
	if !strings.Contains(savedReview.RecommendedNext.Reason, "知识图谱") {
		t.Fatalf("expected knowledge graph reason, got %q", savedReview.RecommendedNext.Reason)
	}
	if !strings.Contains(savedReview.RecommendedNext.Reason, "cache_consistency") {
		t.Fatalf("expected weakest concept to appear in reason, got %q", savedReview.RecommendedNext.Reason)
	}
}

func TestPersistReviewBackfillsLearningPathFromKnowledgeGraphWhenReviewIsSparse(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.EnsureKnowledgeSeeds(ctx); err != nil {
		t.Fatalf("EnsureKnowledgeSeeds() error = %v", err)
	}

	seedSubgraph, err := store.GetKnowledgeSubgraph(ctx, domain.BasicsTopicRedis, "", 2)
	if err != nil {
		t.Fatalf("GetKnowledgeSubgraph() seed error = %v", err)
	}
	if len(seedSubgraph.Nodes) == 0 {
		t.Fatal("expected redis topic seed node")
	}
	root := seedSubgraph.Nodes[0]
	if err := store.UpsertKnowledgeNodes(ctx, "sess_graph_path_seed", []domain.KnowledgeUpdate{
		{
			ScopeType:   domain.MemoryScopeGlobal,
			ParentID:    root.ID,
			Label:       "cache_consistency",
			NodeType:    domain.KnowledgeNodeTypeConcept,
			Proficiency: 1.2,
			Confidence:  0.85,
			Evidence:    "这轮仍然讲不清缓存一致性 trade-off。",
		},
	}); err != nil {
		t.Fatalf("UpsertKnowledgeNodes() error = %v", err)
	}

	session := &domain.TrainingSession{
		ID:          "sess_review_learning_path",
		Mode:        domain.ModeBasics,
		Topic:       domain.BasicsTopicRedis,
		PromptSetID: "stable-v1",
		Intensity:   "standard",
		Status:      domain.StatusReviewPending,
		MaxTurns:    1,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_review_learning_path",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 如何保证缓存一致性？",
		ExpectedPoints: []string{"一致性目标", "更新顺序"},
		WeaknessHits: []domain.WeaknessHit{{
			Kind:     "detail",
			Label:    "缓存一致性",
			Severity: 0.91,
		}},
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	review := &domain.ReviewCard{
		Overall:      "主线能站住，但关键 trade-off 还是说虚了。",
		TopFix:       "补缓存一致性取舍",
		TopFixReason: "这是当前最影响说服力的短板。",
		Highlights:   []string{"主线清楚"},
		Gaps:         []string{"缓存一致性 trade-off 不够具体"},
	}
	if _, err := svc.persistReview(ctx, session, review, &domain.GenerateReviewSideEffects{}); err != nil {
		t.Fatalf("persistReview() error = %v", err)
	}

	savedReview, err := store.GetReview(ctx, session.ReviewID)
	if err != nil {
		t.Fatalf("GetReview() error = %v", err)
	}
	if savedReview == nil {
		t.Fatal("expected saved review")
	}
	if len(savedReview.SuggestedTopics) == 0 || savedReview.SuggestedTopics[0] != domain.BasicsTopicRedis {
		t.Fatalf("expected suggested topic redis, got %#v", savedReview.SuggestedTopics)
	}
	if len(savedReview.NextTrainingFocus) == 0 {
		t.Fatal("expected next training focus to be backfilled")
	}
	if !strings.Contains(savedReview.NextTrainingFocus[0], "cache consistency") {
		t.Fatalf("expected graph-derived focus, got %#v", savedReview.NextTrainingFocus)
	}
}

func TestPersistReviewInfersPrerequisiteEdgeFromRecommendedNextTopic(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session := &domain.TrainingSession{
		ID:          "sess_review_prerequisite",
		Mode:        domain.ModeBasics,
		Topic:       domain.BasicsTopicKafka,
		PromptSetID: "stable-v1",
		Intensity:   "standard",
		Status:      domain.StatusReviewPending,
		MaxTurns:    1,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_review_prerequisite",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Kafka consumer rebalance 怎么影响稳定性？",
		ExpectedPoints: []string{"rebalance 触发", "消费抖动"},
		WeaknessHits: []domain.WeaknessHit{{
			Kind:     "topic",
			Label:    "Kafka rebalance",
			Severity: 0.88,
		}},
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	review := &domain.ReviewCard{
		Overall:      "Kafka 主线还可以，但网络基础短板已经开始拖累理解。",
		TopFix:       "先补网络层基础",
		TopFixReason: "消费者组抖动和网络行为联系不够清楚。",
		Highlights:   []string{"能说出 rebalance 触发点"},
		Gaps:         []string{"网络基础不够扎实"},
		RecommendedNext: &domain.NextSession{
			Mode:   domain.ModeBasics,
			Topic:  domain.BasicsTopicNetwork,
			Reason: "先把网络基础补上。",
		},
	}
	if _, err := svc.persistReview(ctx, session, review, &domain.GenerateReviewSideEffects{}); err != nil {
		t.Fatalf("persistReview() error = %v", err)
	}

	subgraph, err := store.GetKnowledgeSubgraph(ctx, domain.BasicsTopicNetwork, "", 4)
	if err != nil {
		t.Fatalf("GetKnowledgeSubgraph() error = %v", err)
	}

	labels := make(map[string]string, len(subgraph.Nodes))
	for _, node := range subgraph.Nodes {
		labels[node.Label] = node.ID
	}
	if _, ok := labels[domain.BasicsTopicKafka]; !ok {
		t.Fatalf("expected kafka topic to appear as prerequisite neighbor, got %#v", labels)
	}
	if _, ok := labels[domain.BasicsTopicNetwork]; !ok {
		t.Fatalf("expected network topic to appear as root, got %#v", labels)
	}

	found := false
	for _, edge := range subgraph.Edges {
		if edge.SourceID == labels[domain.BasicsTopicNetwork] &&
			edge.TargetID == labels[domain.BasicsTopicKafka] &&
			edge.EdgeType == domain.KnowledgeEdgePrerequisite {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected prerequisite edge network -> kafka after review persistence")
	}
}

func newAgentContextTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handlePromptSetRequest(t, w, r) {
			return
		}
		if r.URL.Path != "/internal/generate_question" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var request domain.GenerateQuestionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.AgentContext == nil {
			t.Fatal("expected agent_context to be preloaded")
		}
		if request.AgentContext.Profile == nil || request.AgentContext.Profile.TargetRole == "" {
			t.Fatalf("expected profile snapshot in agent_context, got %#v", request.AgentContext.Profile)
		}
		if request.AgentContext.KnowledgeSubgraph == nil || len(request.AgentContext.KnowledgeSubgraph.Nodes) == 0 {
			t.Fatalf("expected knowledge subgraph in agent_context, got %#v", request.AgentContext.KnowledgeSubgraph)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.GenerateQuestionResponse{
			Question:       "Redis 的单线程为什么仍然能扛高并发？",
			ExpectedPoints: []string{"事件循环", "内存访问"},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
}
