package service

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/infra/sqlite"
	"practicehelper/server/internal/repo"
)

var testPromptSets = []domain.PromptSetSummary{
	{
		ID:        "stable-v1",
		Label:     "Stable v1",
		Status:    "stable",
		IsDefault: true,
	},
	{
		ID:     "candidate-v1",
		Label:  "Candidate v1",
		Status: "candidate",
	},
}

func handlePromptSetRequest(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	if r.URL.Path != "/internal/prompt-sets" {
		return false
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(testPromptSets); err != nil {
		t.Fatalf("encode prompt sets: %v", err)
	}
	return true
}

func assertStatusEventNames(t *testing.T, events []domain.StreamEvent, want []string) {
	t.Helper()

	var got []string
	for _, event := range events {
		if event.Type == "status" {
			got = append(got, event.Name)
		}
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected status event count: got %v want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected status event sequence: got %v want %v", got, want)
		}
	}
}

func seedSessionForExport(
	t *testing.T,
	store *repo.Store,
	sessionID string,
	withReview bool,
) *domain.TrainingSession {
	t.Helper()

	ctx := context.Background()
	target := seedReadyJobTarget(t, store)
	startedAt := time.Date(2026, 3, 21, 9, 30, 0, 0, time.UTC)
	endedAt := startedAt.Add(18 * time.Minute)

	status := domain.StatusReviewPending
	reviewID := ""
	if withReview {
		status = domain.StatusCompleted
		reviewID = "review_" + sessionID
	}

	session := &domain.TrainingSession{
		ID:                  sessionID,
		Mode:                domain.ModeBasics,
		Topic:               "redis",
		JobTargetID:         target.ID,
		JobTargetAnalysisID: target.LatestSuccessfulAnalysis.ID,
		Intensity:           "pressure",
		Status:              status,
		MaxTurns:            2,
		TotalScore:          82,
		StartedAt:           &startedAt,
		EndedAt:             &endedAt,
		ReviewID:            reviewID,
	}
	turn1 := &domain.TrainingTurn{
		ID:             "turn_1_" + sessionID,
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "单线程事件循环", "合理的数据结构"},
		Answer:         "核心是内存访问快，再加上单线程减少了线程切换成本。",
		Evaluation: &domain.EvaluationResult{
			Score:            78,
			ScoreBreakdown:   map[string]float64{"准确性": 30, "完整性": 24},
			Headline:         "主结论没问题，但对代价讲得还不够。",
			Strengths:        []string{"先说了核心结论"},
			Gaps:             []string{"没有补多路复用和数据结构"},
			Suggestion:       "补上 IO 模型和常见数据结构，回答会更完整。",
			FollowupIntent:   "确认你是否真的理解性能来自哪里。",
			FollowupQuestion: "如果 QPS 上来后出现 CPU 飙高，你会怎么定位？",
			WeaknessHits: []domain.WeaknessHit{
				{Kind: "topic", Label: "缓存淘汰策略", Severity: 0.72},
			},
		},
	}
	if err := store.CreateSession(ctx, session, turn1); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	turn2 := &domain.TrainingTurn{
		ID:             "turn_2_" + sessionID,
		SessionID:      session.ID,
		TurnIndex:      2,
		Stage:          "followup",
		Question:       "如果 QPS 上来后出现 CPU 飙高，你会怎么定位？",
		ExpectedPoints: []string{"先看慢命令", "区分热点 key", "确认是否存在大 key"},
		Answer:         "我会先看 Redis 慢日志，再排查热点 key 和大 key。",
		Evaluation: &domain.EvaluationResult{
			Score:          86,
			ScoreBreakdown: map[string]float64{"准确性": 32, "完整性": 28},
			Headline:       "排查顺序是对的，但还可以补监控闭环。",
			Strengths:      []string{"排查动作顺序合理"},
			Gaps:           []string{"没有提监控和压测回放"},
			Suggestion:     "补监控指标和压测回放，答案会更像真实线上排障。",
		},
	}
	if err := store.InsertTurn(ctx, turn2); err != nil {
		t.Fatalf("InsertTurn() error = %v", err)
	}

	if withReview {
		review := &domain.ReviewCard{
			ID:                reviewID,
			SessionID:         session.ID,
			Overall:           "整体回答有框架，但还要把线上取舍说实。",
			TopFix:            "把缓存淘汰策略和排障闭环讲完整",
			TopFixReason:      "这是这轮最影响说服力的短板。",
			Highlights:        []string{"举了缓存雪崩的真实场景"},
			Gaps:              []string{"缓存淘汰策略讲得太虚"},
			SuggestedTopics:   []string{"redis", "system_design"},
			NextTrainingFocus: []string{"围绕 Redis 性能与故障排查做一轮专项训练"},
			RecommendedNext: &domain.NextSession{
				Mode:   domain.ModeBasics,
				Topic:  "redis",
				Reason: "先把 Redis 相关薄弱点补齐。",
			},
			RetrievalTrace: &domain.RetrievalTrace{
				GeneratedAt: time.Now().UTC(),
				Topic:       "redis",
				ObservationTrace: &domain.MemoryRetrievalTrace{
					MemoryType:     domain.MemoryTypeObservation,
					Query:          "memory_type=observation\ntopic=redis",
					Strategy:       "memory_index_vector_rerank",
					CandidateCount: 4,
					SelectedCount:  1,
					Hits: []domain.MemoryRetrievalHit{{
						Source:        "memory_index",
						MemoryIndexID: "memidx_export_obs",
						RefTable:      "agent_observations",
						RefID:         "obs_export_1",
						ScopeType:     domain.MemoryScopeProject,
						ScopeID:       "proj_export",
						Topic:         "redis",
						Summary:       "项目里的 Redis 观察",
						RuleScore:     5.2,
						VectorScore:   0.92,
						RerankScore:   0.88,
						FinalScore:    8.12,
						Reason:        "项目 scope 命中；semantic 相似度高；rerank 提升排序；final=8.12",
					}},
				},
			},
			ScoreBreakdown: map[string]float64{"表达清晰度": 78, "完整性": 76},
		}
		if err := store.CreateReview(ctx, review); err != nil {
			t.Fatalf("CreateReview() error = %v", err)
		}
	}

	saved, err := store.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved session")
	}
	return saved
}

func seedReadyJobTarget(t *testing.T, store *repo.Store) *domain.JobTarget {
	t.Helper()

	target, err := store.CreateJobTarget(context.Background(), domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "负责高并发后端服务开发，要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}

	run, err := store.StartJobTargetAnalysis(context.Background(), target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}

	if err := store.CompleteJobTargetAnalysis(context.Background(), target.ID, run.ID, &domain.AnalyzeJobTargetResponse{
		Summary:          "核心是在招能独立推进高并发后端系统的人。",
		MustHaveSkills:   []string{"Go", "Redis", "Kafka"},
		BonusSkills:      []string{"Kubernetes"},
		Responsibilities: []string{"负责核心服务设计"},
		EvaluationFocus:  []string{"缓存一致性", "故障排查闭环"},
	}); err != nil {
		t.Fatalf("CompleteJobTargetAnalysis() error = %v", err)
	}

	saved, err := store.GetJobTarget(context.Background(), target.ID)
	if err != nil {
		t.Fatalf("GetJobTarget() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved job target")
	}
	return saved
}

func markJobTargetAnalysisRunning(t *testing.T, store *repo.Store, target *domain.JobTarget) {
	t.Helper()

	run, err := store.StartJobTargetAnalysis(context.Background(), target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}
	if run.Status != domain.JobTargetAnalysisRunning {
		t.Fatalf("expected running status, got %s", run.Status)
	}
}

func markJobTargetAnalysisFailed(t *testing.T, store *repo.Store, target *domain.JobTarget) {
	t.Helper()

	run, err := store.StartJobTargetAnalysis(context.Background(), target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}
	if err := store.FailJobTargetAnalysis(context.Background(), target.ID, run.ID, "boom"); err != nil {
		t.Fatalf("FailJobTargetAnalysis() error = %v", err)
	}
}

func openTestStore(t *testing.T) (*repo.Store, error) {
	t.Helper()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		return nil, err
	}

	if err := sqlite.Bootstrap(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo.New(db), nil
}
