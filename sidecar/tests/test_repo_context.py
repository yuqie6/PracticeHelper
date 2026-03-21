import subprocess
import sys
from pathlib import Path
from unittest.mock import patch

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.llm_client import ModelClientError
from app.repo_context import (
    _sanitize_output,
    clone_repo,
    maybe_apply_github_token,
    path_importance,
    repo_name_from_url,
    split_text,
)


@pytest.mark.parametrize(
    ("repo_url", "want"),
    [
        ("https://github.com/openai/practicehelper.git", "practicehelper"),
        ("https://github.com/openai/practicehelper/", "practicehelper"),
        ("https://example.com/team/repo", "repo"),
    ],
)
def test_repo_name_from_url(repo_url: str, want: str) -> None:
    assert repo_name_from_url(repo_url) == want


@pytest.mark.parametrize(
    ("repo_url", "token", "want"),
    [
        (
            "https://github.com/openai/practicehelper.git",
            "secret-token",
            "https://secret-token@github.com/openai/practicehelper.git",
        ),
        (
            "https://gitlab.com/openai/practicehelper.git",
            "secret-token",
            "https://gitlab.com/openai/practicehelper.git",
        ),
        (
            "https://github.com/openai/practicehelper.git",
            "",
            "https://github.com/openai/practicehelper.git",
        ),
    ],
)
def test_maybe_apply_github_token(repo_url: str, token: str, want: str) -> None:
    assert maybe_apply_github_token(repo_url, token) == want


def test_split_text_single_chunk() -> None:
    assert split_text("short text", chunk_size=100, overlap=10) == ["short text"]


def test_split_text_overlap() -> None:
    parts = split_text("a" * 2200, chunk_size=1000, overlap=200)

    assert len(parts) == 3
    assert parts[0][-200:] == parts[1][:200]
    assert parts[1][-200:] == parts[2][:200]


@pytest.mark.parametrize(
    ("relative_path", "expected_min"),
    [
        ("README.md", 1.0),
        ("docs/architecture.md", 1.0),
        ("random/file.txt", 0.5),
    ],
)
def test_path_importance(relative_path: str, expected_min: float) -> None:
    assert path_importance(relative_path) >= expected_min


def test_clone_repo_raises_on_failure(tmp_path: Path) -> None:
    with patch(
        "app.repo_context.subprocess.run",
        side_effect=subprocess.CalledProcessError(
            returncode=1,
            cmd=["git", "clone"],
            stderr="fatal: could not read from https://secret-token@github.com/openai/practicehelper.git",
        ),
    ):
        with pytest.raises(ModelClientError, match=r"https://\*\*\*@") as exc_info:
            clone_repo(
                "https://github.com/openai/practicehelper.git",
                tmp_path,
                "secret-token",
            )

    assert "secret-token" not in str(exc_info.value)


def test_clone_repo_timeout(tmp_path: Path) -> None:
    with patch(
        "app.repo_context.subprocess.run",
        side_effect=subprocess.TimeoutExpired(cmd=["git", "clone"], timeout=120),
    ):
        with pytest.raises(ModelClientError, match="timed out after 120s"):
            clone_repo(
                "https://github.com/openai/practicehelper.git",
                tmp_path,
                "secret-token",
            )


def test_sanitize_output_strips_token() -> None:
    sanitized = _sanitize_output(
        "fatal: could not read from https://secret-token@github.com/openai/practicehelper.git"
    )

    assert (
        sanitized == "fatal: could not read from https://***@github.com/openai/practicehelper.git"
    )
