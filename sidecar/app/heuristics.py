from __future__ import annotations

import re
import subprocess
import tempfile
from pathlib import Path
from urllib.parse import urlparse

from app.config import Settings
from app.schemas import (
    AnalyzeRepoRequest,
    AnalyzeRepoResponse,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewRequest,
    RepoChunk,
    ReviewCard,
    WeaknessHit,
)

TEXT_EXTENSIONS = {
    ".go",
    ".md",
    ".py",
    ".sql",
    ".ts",
    ".tsx",
    ".js",
    ".jsx",
    ".json",
    ".yaml",
    ".yml",
    ".toml",
    ".txt",
}

IGNORED_PARTS = {
    ".git",
    "node_modules",
    "dist",
    "out",
    "build",
    ".venv",
    "vendor",
    "__pycache__",
}

TECH_KEYWORDS = {
    "go.mod": "Go",
    "package.json": "Node.js",
    "pnpm-lock.yaml": "pnpm",
    "pyproject.toml": "Python",
    "docker-compose.yml": "Docker Compose",
    "Dockerfile": "Docker",
    ".sql": "SQL",
    ".vue": "Vue",
    ".tsx": "React",
    ".ts": "TypeScript",
    ".py": "Python",
    ".go": "Go",
    ".md": "Markdown",
}


def analyze_repo(request: AnalyzeRepoRequest, settings: Settings) -> AnalyzeRepoResponse:
    with tempfile.TemporaryDirectory(prefix="practicehelper-repo-") as temp_dir:
        repo_dir = clone_repo(request.repo_url, Path(temp_dir), settings.github_token)
        files = collect_text_files(repo_dir)
        chunks = build_repo_chunks(repo_dir, files)

        default_branch = run_git(repo_dir, "rev-parse", "--abbrev-ref", "HEAD")
        import_commit = run_git(repo_dir, "rev-parse", "HEAD")
        project_name = repo_dir.name
        top_paths = [chunk.file_path for chunk in chunks[:5]]
        tech_stack = detect_tech_stack(repo_dir, files)
        summary = build_summary(project_name, tech_stack, top_paths)

        return AnalyzeRepoResponse(
            repo_url=request.repo_url,
            name=project_name,
            default_branch=default_branch or "main",
            import_commit=import_commit,
            summary=summary,
            tech_stack=tech_stack,
            highlights=build_highlights(project_name, chunks),
            challenges=build_challenges(chunks),
            tradeoffs=build_tradeoffs(chunks),
            ownership_points=build_ownership_points(project_name, chunks),
            followup_points=build_followup_points(chunks),
            chunks=chunks[:120],
        )


def generate_question(request: GenerateQuestionRequest) -> GenerateQuestionResponse:
    if request.mode == "project" and request.project is not None:
        expected = [
            "问题背景",
            "技术选型理由",
            "trade-off",
            "落地细节",
            "真实结果",
        ]
        focus = request.project.followup_points[:2] or request.project.highlights[:2]
        if focus:
            expected.extend(focus)
        question = (
            f"请你像面试现场一样介绍 {request.project.name}，重点讲清楚为什么这样设计，"
            f"以及你自己真正扛下来的部分。"
        )
        return GenerateQuestionResponse(question=question, expected_points=dedupe(expected)[:6])

    template = request.templates[0] if request.templates else None
    if template is None:
        prompt = f"请系统回答 {request.topic} 这个主题里你最容易被追问的一道题。"
        return GenerateQuestionResponse(
            question=prompt,
            expected_points=["定义", "原理", "落地场景", "风险点"],
        )

    return GenerateQuestionResponse(
        question=template.prompt,
        expected_points=template.focus_points[:5],
    )


def evaluate_answer(request: EvaluateAnswerRequest) -> EvaluationResult:
    answer = normalize_text(request.answer)
    expected = request.expected_points or ["定义", "原理", "落地场景"]

    matched_points = []
    missing_points = []
    for point in expected:
        if point_matches_answer(point, answer):
            matched_points.append(point)
        else:
            missing_points.append(point)

    completeness_ratio = len(matched_points) / max(len(expected), 1)
    length_ratio = min(len(answer) / 220, 1.0)
    grounding_ratio = grounding_score(answer)
    clarity_ratio = clarity_score(answer)

    breakdown = build_score_breakdown(
        request.mode,
        completeness_ratio=completeness_ratio,
        length_ratio=length_ratio,
        grounding_ratio=grounding_ratio,
        clarity_ratio=clarity_ratio,
        followup_ratio=0.7 if request.is_followup else 0.55,
    )

    score = round(sum(breakdown.values()), 1)
    strengths = build_strengths(matched_points, answer)
    gaps = build_gaps(missing_points, answer)
    weakness_hits = build_weakness_hits(request, score, answer, missing_points)

    followup_question = ""
    followup_expected_points: list[str] = []
    if not request.is_followup:
        followup_target = missing_points[0] if missing_points else expected[0]
        followup_question = build_followup_question(request, followup_target)
        followup_expected_points = [followup_target, "具体方案", "踩坑与修正"]

    return EvaluationResult(
        score=score,
        score_breakdown=breakdown,
        strengths=strengths,
        gaps=gaps,
        followup_question=followup_question,
        followup_expected_points=followup_expected_points,
        weakness_hits=weakness_hits,
    )


def generate_review(request: GenerateReviewRequest) -> ReviewCard:
    evaluations = []
    for turn in request.turns:
        if turn.evaluation is not None:
            evaluations.append(turn.evaluation)
        if turn.followup_evaluation is not None:
            evaluations.append(turn.followup_evaluation)

    if not evaluations:
        return ReviewCard(
            overall="这一轮没有拿到有效评估结果，先补完回答再看复盘。",
            highlights=[],
            gaps=["没有形成完整回答"],
            suggested_topics=["先完成一轮完整训练"],
            next_training_focus=["补完整回答"],
            score_breakdown={},
        )

    merged_breakdown = merge_breakdowns([item.score_breakdown for item in evaluations])
    merged_highlights = dedupe([text for item in evaluations for text in item.strengths])[:4]
    merged_gaps = dedupe([text for item in evaluations for text in item.gaps])[:5]
    average_score = round(sum(item.score for item in evaluations) / len(evaluations), 1)

    if request.session.mode == "project" and request.project is not None:
        overall = (
            f"{request.project.name} 这一轮平均分 {average_score}。整体能讲出主线，"
            "但需要把 ownership、trade-off 和真实结果讲得更硬。"
        )
        suggested = [request.project.name, "项目表达", "架构权衡"]
    else:
        topic = request.session.topic or "基础主题"
        overall = (
            f"{topic} 这一轮平均分 {average_score}。"
            "基础点能覆盖一部分，但追问下的细节支撑还不够稳。"
        )
        suggested = [topic, "追问抗压", "落地细节"]

    return ReviewCard(
        overall=overall,
        highlights=merged_highlights,
        gaps=merged_gaps,
        suggested_topics=dedupe(suggested + merged_gaps)[:4],
        next_training_focus=build_next_training_focus(request, merged_gaps),
        score_breakdown=merged_breakdown,
    )


def clone_repo(repo_url: str, temp_root: Path, github_token: str) -> Path:
    repo_dir = temp_root / repo_name_from_url(repo_url)
    auth_url = maybe_apply_github_token(repo_url, github_token)
    subprocess.run(
        ["git", "clone", "--depth", "1", auth_url, str(repo_dir)],
        check=True,
        capture_output=True,
        text=True,
    )
    return repo_dir


def maybe_apply_github_token(repo_url: str, github_token: str) -> str:
    if not github_token:
        return repo_url
    parsed = urlparse(repo_url)
    if parsed.scheme != "https" or parsed.hostname != "github.com":
        return repo_url
    return f"https://{github_token}@github.com{parsed.path}"


def repo_name_from_url(repo_url: str) -> str:
    tail = repo_url.rstrip("/").split("/")[-1]
    return tail[:-4] if tail.endswith(".git") else tail


def run_git(repo_dir: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", "-C", str(repo_dir), *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def collect_text_files(repo_dir: Path) -> list[Path]:
    files: list[Path] = []
    for path in repo_dir.rglob("*"):
        if not path.is_file():
            continue
        if any(part in IGNORED_PARTS for part in path.parts):
            continue
        if path.suffix.lower() not in TEXT_EXTENSIONS and path.name not in TECH_KEYWORDS:
            continue
        if path.stat().st_size > 256_000:
            continue
        files.append(path)
    files.sort(key=lambda item: (path_priority(repo_dir, item), str(item)))
    return files[:80]


def build_repo_chunks(repo_dir: Path, files: list[Path]) -> list[RepoChunk]:
    chunks: list[RepoChunk] = []
    for path in files:
        text = path.read_text(encoding="utf-8", errors="ignore").strip()
        if not text:
            continue
        relative = path.relative_to(repo_dir).as_posix()
        parts = split_text(text)
        importance = path_importance(relative)
        for index, part in enumerate(parts):
            chunks.append(
                RepoChunk(
                    file_path=relative,
                    file_type=path.suffix.lower() or path.name,
                    content=part,
                    importance=importance,
                    fts_key=f"{relative}#{index}",
                )
            )
    chunks.sort(key=lambda item: (-item.importance, item.file_path))
    return chunks


def split_text(text: str, chunk_size: int = 1400, overlap: int = 220) -> list[str]:
    normalized = re.sub(r"\n{3,}", "\n\n", text)
    if len(normalized) <= chunk_size:
        return [normalized]

    parts: list[str] = []
    start = 0
    while start < len(normalized):
        end = min(len(normalized), start + chunk_size)
        parts.append(normalized[start:end])
        if end == len(normalized):
            break
        start = end - overlap
    return parts


def detect_tech_stack(repo_dir: Path, files: list[Path]) -> list[str]:
    detected: list[str] = []
    for path in files:
        relative = path.relative_to(repo_dir).as_posix()
        if relative in TECH_KEYWORDS:
            detected.append(TECH_KEYWORDS[relative])
        suffix = path.suffix.lower()
        if suffix in TECH_KEYWORDS:
            detected.append(TECH_KEYWORDS[suffix])
    return dedupe(detected)[:8]


def build_summary(project_name: str, tech_stack: list[str], top_paths: list[str]) -> str:
    stack_text = "、".join(tech_stack[:4]) if tech_stack else "多语言工程栈"
    path_text = "、".join(top_paths[:3]) if top_paths else "核心文档和入口文件"
    return (
        f"{project_name} 是一个以 {stack_text} 为主的工程项目，"
        f"仓库里最关键的材料集中在 {path_text}。"
    )


def build_highlights(project_name: str, chunks: list[RepoChunk]) -> list[str]:
    top_paths = [chunk.file_path for chunk in chunks[:4]]
    highlights = [
        f"{project_name} 已经有可用于面试训练的真实仓库材料",
        "仓库内同时包含文档层和实现层线索，适合做项目深挖",
    ]
    highlights.extend(f"关键路径可从 {path} 展开" for path in top_paths[:2])
    return dedupe(highlights)[:4]


def build_challenges(chunks: list[RepoChunk]) -> list[str]:
    challenge_keywords = []
    for chunk in chunks[:10]:
        lowered = chunk.content.lower()
        if "retry" in lowered or "重试" in chunk.content:
            challenge_keywords.append("失败重试与状态一致性")
        if "cache" in lowered or "redis" in lowered:
            challenge_keywords.append("缓存一致性与热点数据管理")
        if "queue" in lowered or "kafka" in lowered:
            challenge_keywords.append("异步链路与幂等处理")
    challenge_keywords.extend(["技术选型理由是否足够具体", "线上问题和复盘是否说得出来"])
    return dedupe(challenge_keywords)[:4]


def build_tradeoffs(chunks: list[RepoChunk]) -> list[str]:
    items = ["为什么选当前技术栈而不是更重的方案", "实现速度和工程治理之间怎么权衡"]
    if any("sqlite" in chunk.content.lower() for chunk in chunks):
        items.append("本地存储和可扩展性之间的取舍")
    if any(
        "agent" in chunk.content.lower() or "langgraph" in chunk.content.lower() for chunk in chunks
    ):
        items.append("为什么用受控编排，而不是复杂多 agent")
    return dedupe(items)[:4]


def build_ownership_points(project_name: str, chunks: list[RepoChunk]) -> list[str]:
    items = [
        f"你在 {project_name} 里真正负责的模块是什么",
        "你做过哪些关键决策，而不是只做实现",
        "你遇到过什么坑，最后怎么修",
    ]
    if chunks:
        items.append(f"你为什么优先从 {chunks[0].file_path} 这条主线切入")
    return dedupe(items)[:4]


def build_followup_points(chunks: list[RepoChunk]) -> list[str]:
    points = ["架构主线", "技术选型", "难点处理", "trade-off", "故障复盘"]
    for chunk in chunks[:8]:
        lowered = chunk.content.lower()
        if "api" in lowered:
            points.append("接口边界")
        if "worker" in lowered or "job" in lowered:
            points.append("异步任务设计")
        if "memory" in lowered:
            points.append("记忆与上下文设计")
    return dedupe(points)[:6]


def build_score_breakdown(
    mode: str,
    *,
    completeness_ratio: float,
    length_ratio: float,
    grounding_ratio: float,
    clarity_ratio: float,
    followup_ratio: float,
) -> dict[str, float]:
    if mode == "project":
        return {
            "ownership": round(25 * grounding_ratio, 1),
            "落地感": round(20 * grounding_ratio, 1),
            "trade-off 清晰度": round(20 * completeness_ratio, 1),
            "完整性": round(15 * completeness_ratio, 1),
            "表达清晰度": round(10 * clarity_ratio, 1),
            "抗追问能力": round(10 * followup_ratio, 1),
        }

    return {
        "准确性": round(30 * completeness_ratio, 1),
        "完整性": round(25 * completeness_ratio, 1),
        "落地感": round(15 * grounding_ratio, 1),
        "表达清晰度": round(15 * clarity_ratio, 1),
        "抗追问能力": round(15 * max(length_ratio, followup_ratio), 1),
    }


def build_strengths(matched_points: list[str], answer: str) -> list[str]:
    strengths: list[str] = []
    if matched_points:
        strengths.append(f"覆盖到了这些关键点：{'、'.join(matched_points[:3])}")
    if grounding_score(answer) >= 0.6:
        strengths.append("回答里有一定落地感，不是纯空话")
    if len(answer) >= 180:
        strengths.append("回答展开度还可以，没有只给一句话答案")
    return dedupe(strengths)[:3]


def build_gaps(missing_points: list[str], answer: str) -> list[str]:
    gaps: list[str] = []
    if missing_points:
        gaps.append(f"这些点讲得不够：{'、'.join(missing_points[:3])}")
    if len(answer) < 120:
        gaps.append("回答太短，容易一追问就塌")
    if clarity_score(answer) < 0.45:
        gaps.append("表达顺序有点乱，主线不够清楚")
    return dedupe(gaps)[:4]


def build_followup_question(request: EvaluateAnswerRequest, followup_target: str) -> str:
    if request.mode == "project" and request.project is not None:
        return (
            f"你刚才提到 {followup_target} 还不够硬。请结合 {request.project.name} 里的真实实现，"
            "讲清楚你怎么做、为什么这么做、最后效果怎样。"
        )
    return f"你刚才对 {followup_target} 讲得比较虚，能不能结合真实场景把原理和落地再讲具体一点？"


def build_weakness_hits(
    request: EvaluateAnswerRequest,
    score: float,
    answer: str,
    missing_points: list[str],
) -> list[WeaknessHit]:
    hits: list[WeaknessHit] = []
    if request.mode == "project" and request.project is not None and score < 75:
        hits.append(WeaknessHit(kind="project", label=request.project.name, severity=0.65))
    if request.mode == "basics" and request.topic and score < 75:
        hits.append(WeaknessHit(kind="topic", label=request.topic, severity=0.6))
    if len(answer) < 140 or clarity_score(answer) < 0.45:
        hits.append(WeaknessHit(kind="expression", label="表达不够展开", severity=0.5))
    if request.is_followup and missing_points:
        hits.append(WeaknessHit(kind="followup_breakdown", label=missing_points[0], severity=0.6))
    return hits


def build_next_training_focus(request: GenerateReviewRequest, gaps: list[str]) -> list[str]:
    if request.session.mode == "project" and request.project is not None:
        base = [f"{request.project.name} 项目专项", "ownership 表达", "trade-off 说明"]
    else:
        base = [f"{request.session.topic or '基础主题'} 专项", "追问抗压", "落地细节"]
    return dedupe(base + gaps)[:4]


def merge_breakdowns(items: list[dict[str, float]]) -> dict[str, float]:
    merged: dict[str, float] = {}
    for item in items:
        for key, value in item.items():
            merged[key] = merged.get(key, 0.0) + value
    if not items:
        return merged
    return {key: round(value / len(items), 1) for key, value in merged.items()}


def normalize_text(text: str) -> str:
    return re.sub(r"\s+", " ", text).strip().lower()


def point_matches_answer(point: str, answer: str) -> bool:
    lowered = normalize_text(point)
    tokens = [
        token for token in re.split(r"[^a-zA-Z0-9\u4e00-\u9fff]+", lowered) if len(token) >= 2
    ]
    if not tokens:
        return lowered in answer
    return any(token in answer for token in tokens)


def grounding_score(answer: str) -> float:
    markers = ["因为", "所以", "例如", "比如", "线上", "实现", "权衡", "trade-off", "方案"]
    hits = sum(1 for marker in markers if marker in answer)
    return min(hits / 5, 1.0)


def clarity_score(answer: str) -> float:
    markers = ["首先", "然后", "最后", "一方面", "另一方面", "先", "再", "最后"]
    hits = sum(1 for marker in markers if marker in answer)
    return min((hits + (1 if len(answer) >= 160 else 0)) / 4, 1.0)


def path_priority(repo_dir: Path, path: Path) -> tuple[int, int]:
    relative = path.relative_to(repo_dir).as_posix().lower()
    return (-int(path_importance(relative) * 100), len(relative))


def path_importance(relative: str) -> float:
    lowered = relative.lower()
    if "readme" in lowered or "architecture" in lowered or "prd" in lowered or "plan" in lowered:
        return 1.1
    if lowered.endswith(("go.mod", "package.json", "pyproject.toml", "docker-compose.yml")):
        return 0.95
    if "/cmd/" in lowered or "/src/" in lowered or "/app/" in lowered:
        return 0.8
    return 0.55


def dedupe(items: list[str]) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for item in items:
        normalized = item.strip()
        if not normalized or normalized in seen:
            continue
        seen.add(normalized)
        result.append(normalized)
    return result
