package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
