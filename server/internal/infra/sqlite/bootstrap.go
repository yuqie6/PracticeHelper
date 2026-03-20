package sqlite

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"practicehelper/server/internal/domain"
)

func Bootstrap(db *sql.DB) error {
	if err := migrate(db); err != nil {
		return err
	}

	if err := seedQuestionTemplates(db); err != nil {
		return err
	}

	return nil
}

func migrate(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS user_profile (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			target_role TEXT NOT NULL,
			target_company_type TEXT NOT NULL,
			current_stage TEXT NOT NULL,
			application_deadline TEXT NOT NULL DEFAULT '',
			tech_stacks_json TEXT NOT NULL,
			primary_projects_json TEXT NOT NULL,
			self_reported_weaknesses_json TEXT NOT NULL,
			active_job_target_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS project_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			repo_url TEXT NOT NULL UNIQUE,
			default_branch TEXT NOT NULL,
			import_commit TEXT NOT NULL,
			summary TEXT NOT NULL,
			tech_stack_json TEXT NOT NULL,
			highlights_json TEXT NOT NULL,
			challenges_json TEXT NOT NULL,
			tradeoffs_json TEXT NOT NULL,
			ownership_points_json TEXT NOT NULL,
			followup_points_json TEXT NOT NULL,
			import_status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS project_import_jobs (
			id TEXT PRIMARY KEY,
			repo_url TEXT NOT NULL,
			status TEXT NOT NULL,
			stage TEXT NOT NULL,
			message TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			project_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT NOT NULL DEFAULT '',
			finished_at TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS job_targets (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			company_name TEXT NOT NULL DEFAULT '',
			source_text TEXT NOT NULL,
			latest_analysis_id TEXT NOT NULL DEFAULT '',
			latest_analysis_status TEXT NOT NULL DEFAULT 'idle',
			last_used_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS job_target_analysis_runs (
			id TEXT PRIMARY KEY,
			job_target_id TEXT NOT NULL REFERENCES job_targets(id) ON DELETE CASCADE,
			source_text_snapshot TEXT NOT NULL,
			status TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			must_have_skills_json TEXT NOT NULL DEFAULT '[]',
			bonus_skills_json TEXT NOT NULL DEFAULT '[]',
			responsibilities_json TEXT NOT NULL DEFAULT '[]',
			evaluation_focus_json TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS repo_chunks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES project_profiles(id) ON DELETE CASCADE,
			file_path TEXT NOT NULL,
			file_type TEXT NOT NULL,
			content TEXT NOT NULL,
			importance REAL NOT NULL,
			fts_key TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS repo_chunks_fts USING fts5(
			chunk_id UNINDEXED,
			project_id UNINDEXED,
			file_path,
			file_type,
			content
		);`,
		`CREATE TABLE IF NOT EXISTS question_templates (
			id TEXT PRIMARY KEY,
			mode TEXT NOT NULL,
			topic TEXT NOT NULL,
			prompt TEXT NOT NULL,
			focus_points_json TEXT NOT NULL,
			bad_answers_json TEXT NOT NULL,
			followup_templates_json TEXT NOT NULL,
			score_weights_json TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS training_sessions (
			id TEXT PRIMARY KEY,
			mode TEXT NOT NULL,
			topic TEXT NOT NULL DEFAULT '',
			project_id TEXT NOT NULL DEFAULT '',
			job_target_id TEXT NOT NULL DEFAULT '',
			job_target_analysis_id TEXT NOT NULL DEFAULT '',
			intensity TEXT NOT NULL,
			status TEXT NOT NULL,
			total_score REAL NOT NULL DEFAULT 0,
			started_at TEXT NOT NULL DEFAULT '',
			ended_at TEXT NOT NULL DEFAULT '',
			review_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS training_turns (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES training_sessions(id) ON DELETE CASCADE,
			turn_index INTEGER NOT NULL,
			stage TEXT NOT NULL,
			question TEXT NOT NULL,
			expected_points_json TEXT NOT NULL,
			answer TEXT NOT NULL DEFAULT '',
			evaluation_json TEXT NOT NULL DEFAULT '{}',
			followup_question TEXT NOT NULL DEFAULT '',
			followup_expected_points_json TEXT NOT NULL DEFAULT '[]',
			followup_answer TEXT NOT NULL DEFAULT '',
			followup_evaluation_json TEXT NOT NULL DEFAULT '{}',
			weakness_hits_json TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS review_cards (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL UNIQUE REFERENCES training_sessions(id) ON DELETE CASCADE,
			overall TEXT NOT NULL,
			highlights_json TEXT NOT NULL,
			gaps_json TEXT NOT NULL,
			suggested_topics_json TEXT NOT NULL,
			next_training_focus_json TEXT NOT NULL,
			score_breakdown_json TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS weakness_tags (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			label TEXT NOT NULL,
			severity REAL NOT NULL,
			frequency INTEGER NOT NULL,
			last_seen_at TEXT NOT NULL,
			evidence_session_id TEXT NOT NULL,
			UNIQUE(kind, label)
		);`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("run migration: %w", err)
		}
	}

	if err := ensureColumn(db, "review_cards", "top_fix", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "review_cards", "top_fix_reason", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "review_cards", "recommended_next_json", "TEXT NOT NULL DEFAULT 'null'"); err != nil {
		return err
	}
	if err := ensureColumn(db, "training_sessions", "job_target_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "training_sessions", "job_target_analysis_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "user_profile", "active_job_target_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	return nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("query table info for %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return fmt.Errorf("scan table info for %s: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info for %s: %w", table, err)
	}

	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}

func seedQuestionTemplates(db *sql.DB) error {
	templates := []domain.QuestionTemplate{
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "go",
			Prompt:            "Go 的 goroutine 为什么比系统线程更轻量？",
			FocusPoints:       []string{"GMP 调度", "栈扩缩容", "协作式与抢占式调度", "上下文切换成本"},
			BadAnswers:        []string{"只说 goroutine 很轻，不解释为什么", "只背概念，不提调度器和栈"},
			FollowupTemplates: []string{"如果 goroutine 很多，调度器怎么避免完全失控？", "goroutine 泄漏在线上怎么排查？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "go",
			Prompt:            "Go 的 channel 和 mutex 各适合什么场景？",
			FocusPoints:       []string{"共享状态保护", "所有权转移", "性能与复杂度取舍", "典型误用"},
			BadAnswers:        []string{"只说 channel 更高级", "只说 mutex 更快，不说场景边界"},
			FollowupTemplates: []string{"什么时候为了不用 mutex 而硬上 channel 会更糟？", "你在线上怎么判断这段并发逻辑该不该重构？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "go",
			Prompt:            "Go 里的 context 一般解决什么问题？什么时候会被滥用？",
			FocusPoints:       []string{"取消与超时", "跨边界透传", "请求级元数据", "不要塞业务参数"},
			BadAnswers:        []string{"只说传参", "只说用来控制超时，不提边界和滥用"},
			FollowupTemplates: []string{"如果某个下游不响应，context 应该怎么配合止血？", "为什么把可选业务参数塞进 context 是坏味道？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "go",
			Prompt:            "接口很稳定时，你还会在 Go 里保留 interface 抽象吗？",
			FocusPoints:       []string{"面向变化点抽象", "测试替身", "包边界", "过度抽象成本"},
			BadAnswers:        []string{"只说 interface 解耦", "把所有依赖都抽象成接口"},
			FollowupTemplates: []string{"什么时候应该先写具体类型，后面再抽接口？", "为了测试而抽接口时，怎么避免抽象失真？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "go",
			Prompt:            "在 Go 服务里遇到内存持续上涨，你会怎么排查？",
			FocusPoints:       []string{"pprof", "堆对象定位", "逃逸与缓存", "泄漏和正常增长区分", "回归验证"},
			BadAnswers:        []string{"只说看 top", "只说加机器，不说定位过程"},
			FollowupTemplates: []string{"如果堆不高但 RSS 一直涨，你会继续看哪里？", "修完后怎么确认不是靠运气消失？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "redis",
			Prompt:            "Redis 为什么快？",
			FocusPoints:       []string{"内存访问", "事件循环", "高效数据结构", "网络模型", "避免大 key"},
			BadAnswers:        []string{"只说因为在内存里", "只说单线程快，不说 IO 模型和数据结构"},
			FollowupTemplates: []string{"Redis 6 的多线程多在哪里？", "大 key 和持久化会带来什么问题？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "redis",
			Prompt:            "缓存击穿、缓存穿透、缓存雪崩怎么区分？分别怎么兜底？",
			FocusPoints:       []string{"问题定义", "热点 key", "空值缓存/布隆过滤器", "过期打散", "降级思路"},
			BadAnswers:        []string{"三个概念混着说", "只说加锁，不说不同场景的根因"},
			FollowupTemplates: []string{"热点 key 真被打爆时，只加互斥锁够吗？", "空值缓存要注意什么副作用？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "redis",
			Prompt:            "Redis 持久化里 RDB 和 AOF 怎么取舍？",
			FocusPoints:       []string{"恢复速度", "数据丢失窗口", "磁盘与 CPU 开销", "混合持久化", "业务场景"},
			BadAnswers:        []string{"只背 RDB 快 AOF 安全", "不谈线上恢复和资源代价"},
			FollowupTemplates: []string{"如果重启时间很敏感，你会怎么配置持久化？", "AOF 重写时要注意什么抖动风险？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "redis",
			Prompt:            "Redis 分布式锁为什么不能只靠 SETNX？",
			FocusPoints:       []string{"过期时间", "误删锁", "原子释放", "续期", "锁的适用边界"},
			BadAnswers:        []string{"只说 SETNX 就够了", "只背 Redlock，不说单实例实现的底线要求"},
			FollowupTemplates: []string{"业务执行时间超过锁过期时间时你怎么处理？", "什么场景下你宁愿不用 Redis 锁？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "redis",
			Prompt:            "你会怎么发现并治理 Redis 大 key？",
			FocusPoints:       []string{"识别方式", "业务影响", "拆分策略", "渐进迁移", "监控验证"},
			BadAnswers:        []string{"只说删掉", "只说会慢，不说具体影响和迁移风险"},
			FollowupTemplates: []string{"如果这个大 key 不能直接删，你会怎么平滑治理？", "为什么大 key 问题不只是慢查询？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "kafka",
			Prompt:            "Kafka 消费幂等一般怎么做？",
			FocusPoints:       []string{"业务幂等键", "重复消费", "事务边界", "offset 提交时机", "失败重试策略"},
			BadAnswers:        []string{"只说 Kafka 自己保证", "只背 at least once，不落到业务实现"},
			FollowupTemplates: []string{"为什么 offset 不能先提交？", "如果重试和 DLQ 设计不好会怎样？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "kafka",
			Prompt:            "Kafka 为什么能做到高吞吐？代价是什么？",
			FocusPoints:       []string{"顺序写磁盘", "批量压缩", "零拷贝", "分区并行", "延迟与一致性代价"},
			BadAnswers:        []string{"只说因为顺序写", "只讲快，不讲代价和适用前提"},
			FollowupTemplates: []string{"吞吐上去了，为什么尾延迟不一定好看？", "分区加多之后消费者一定更快吗？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "kafka",
			Prompt:            "Kafka 里消息积压了，你会怎么定位根因？",
			FocusPoints:       []string{"生产与消费速率对比", "分区倾斜", "下游瓶颈", "rebalance", "止血方案"},
			BadAnswers:        []string{"只说扩容消费者", "不区分 broker、consumer、下游服务的问题"},
			FollowupTemplates: []string{"如果扩消费者也没用，你下一步看什么？", "积压消掉以后怎么防止再次发生？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "kafka",
			Prompt:            "一个 topic 该怎么规划 partition 数？",
			FocusPoints:       []string{"吞吐需求", "consumer 并行度", "顺序要求", "运维成本", "扩分区副作用"},
			BadAnswers:        []string{"越多越好", "只看消费者数量，不谈顺序和资源成本"},
			FollowupTemplates: []string{"为什么扩 partition 可能反而带来更多问题？", "顺序消费要求强时你会怎么折中？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "kafka",
			Prompt:            "Kafka 消费失败时，重试队列和 DLQ 应该怎么设计？",
			FocusPoints:       []string{"瞬时失败与永久失败区分", "重试节奏", "幂等", "可观测性", "人工介入路径"},
			BadAnswers:        []string{"失败就一直重试", "只有 DLQ，没有回放和排障路径"},
			FollowupTemplates: []string{"为什么同分区里的失败消息不能简单跳过去？", "DLQ 放久了没人看，系统上会出什么假象？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
	}

	for _, template := range templates {
		result, err := db.Exec(
			`INSERT INTO question_templates (
				id, mode, topic, prompt, focus_points_json, bad_answers_json, followup_templates_json, score_weights_json
			)
			SELECT ?, ?, ?, ?, ?, ?, ?, ?
			WHERE NOT EXISTS (
				SELECT 1
				FROM question_templates
				WHERE mode = ? AND topic = ? AND prompt = ?
			)`,
			template.ID,
			template.Mode,
			template.Topic,
			template.Prompt,
			mustJSON(template.FocusPoints),
			mustJSON(template.BadAnswers),
			mustJSON(template.FollowupTemplates),
			mustJSON(template.ScoreWeights),
			template.Mode,
			template.Topic,
			template.Prompt,
		)
		if err != nil {
			return fmt.Errorf("insert question template: %w", err)
		}
		if affected, err := result.RowsAffected(); err != nil {
			return fmt.Errorf("count inserted question template rows: %w", err)
		} else if affected > 1 {
			return fmt.Errorf("unexpected inserted rows for question template %s/%s", template.Topic, template.Prompt)
		}
	}

	return nil
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func newID(prefix string) string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buffer))
}
