from __future__ import annotations

import re
import subprocess
import tempfile
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import urlparse

from app.config import Settings
from app.schemas import AnalyzeRepoRequest, RepoChunk

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


@dataclass(frozen=True)
class RepoAnalysisBundle:
    repo_url: str
    name: str
    default_branch: str
    import_commit: str
    tech_stack: list[str]
    top_paths: list[str]
    chunks: list[RepoChunk]


def collect_repo_analysis_bundle(
    request: AnalyzeRepoRequest,
    settings: Settings,
) -> RepoAnalysisBundle:
    # 仓库导入走的是“低成本一次性快照”而不是全量索引：
    # 临时 clone、限制文件数和 chunk 数，优先保证导入时延与 token 成本可控。
    with tempfile.TemporaryDirectory(prefix="practicehelper-repo-") as temp_dir:
        repo_dir = clone_repo(request.repo_url, Path(temp_dir), settings.github_token)
        files = collect_text_files(repo_dir)
        chunks = build_repo_chunks(repo_dir, files)
        return RepoAnalysisBundle(
            repo_url=request.repo_url,
            name=repo_dir.name,
            default_branch=run_git(repo_dir, "rev-parse", "--abbrev-ref", "HEAD") or "main",
            import_commit=run_git(repo_dir, "rev-parse", "HEAD"),
            tech_stack=detect_tech_stack(repo_dir, files),
            top_paths=[chunk.file_path for chunk in chunks[:8]],
            chunks=chunks[:120],
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

    # 保留 overlap 不是为了追求最大召回，而是尽量减少摘要或追问时
    # 因为硬切分导致的上下文断裂，让相邻 chunk 至少共享一段过渡语义。
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


def path_priority(repo_dir: Path, path: Path) -> tuple[int, int]:
    relative = path.relative_to(repo_dir).as_posix().lower()
    return (-int(path_importance(relative) * 100), len(relative))


def path_importance(relative_path: str) -> float:
    lowered = relative_path.lower()
    score = 0.5
    if lowered.startswith(("readme", "docs/", "architecture", "plan", "prd")):
        score += 0.5
    if any(part in lowered for part in ("/cmd/", "/internal/", "/app/", "/src/")):
        score += 0.35
    if lowered.endswith(
        ("main.go", "main.py", "router.ts", "package.json", "go.mod", "pyproject.toml")
    ):
        score += 0.4
    return min(score, 1.5)


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
