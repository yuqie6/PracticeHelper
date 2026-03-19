import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.agent_runtime import AgentRuntime
from app.config import Settings
from app.llm_client import ChatCompletionResult
from app.schemas import (
    EvaluateAnswerRequest,
    GenerateQuestionRequest,
    ProjectProfile,
)


class FakeModelClient:
    def __init__(self, responses: list[ChatCompletionResult]) -> None:
        self._responses = responses
        self.calls: list[dict] = []

    def create_completion(self, *, messages, tools=None, temperature=0.2):
        self.calls.append({"messages": messages, "tools": tools, "temperature": temperature})
        if not self._responses:
            raise AssertionError("fake client has no more responses")
        return self._responses.pop(0)


def test_generate_question_uses_tool_loop_before_returning_json() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeModelClient(
            [
                ChatCompletionResult(
                    content="",
                    tool_calls=[
                        {
                            "id": "call_templates",
                            "function": {
                                "name": "read_question_templates",
                                "arguments": "{}",
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"question":"请讲讲 Mirror 的 trade-off。","expected_points":'
                        '["问题背景","技术选型理由","trade-off","真实结果"]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    response = runtime.generate_question(
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


def test_runtime_falls_back_to_heuristics_when_llm_is_disabled() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="",
            openai_base_url="",
            openai_api_key="",
            llm_timeout_seconds=10,
        )
    )

    result = runtime.evaluate_answer(
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
