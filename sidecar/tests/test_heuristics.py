import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.heuristics import evaluate_answer, generate_question, generate_review
from app.schemas import (
    EvaluateAnswerRequest,
    GenerateQuestionRequest,
    GenerateReviewRequest,
    ProjectProfile,
    TrainingSession,
    TrainingTurn,
)


def test_generate_question_for_project_uses_project_name() -> None:
    response = generate_question(
        GenerateQuestionRequest(
            mode="project",
            intensity="standard",
            project=ProjectProfile(
                name="Mirror",
                summary="Agent workflow",
                followup_points=["trade-off"],
            ),
        )
    )

    assert "Mirror" in response.question
    assert "trade-off" in response.expected_points


def test_evaluate_answer_marks_short_answers_as_gap() -> None:
    result = evaluate_answer(
        EvaluateAnswerRequest(
            mode="basics",
            topic="redis",
            question="Redis 为什么快？",
            expected_points=["内存访问", "事件循环", "高效数据结构"],
            answer="因为它在内存里。",
        )
    )

    assert result.score < 70
    assert result.gaps


def test_generate_review_aggregates_turns() -> None:
    evaluation = evaluate_answer(
        EvaluateAnswerRequest(
            mode="basics",
            topic="go",
            question="goroutine 为什么轻量？",
            expected_points=["GMP 调度", "栈扩缩容", "上下文切换成本"],
            answer="首先它有 GMP 调度，其次栈可以扩缩容，所以切换成本比线程低。",
        )
    )
    turn = TrainingTurn(
        question="goroutine 为什么轻量？",
        expected_points=["GMP 调度"],
        evaluation=evaluation,
    )
    review = generate_review(
        GenerateReviewRequest(
            session=TrainingSession(mode="basics", topic="go", intensity="standard"),
            turns=[turn],
        )
    )

    assert review.overall
    assert review.score_breakdown
