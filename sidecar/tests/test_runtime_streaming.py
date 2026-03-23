# ruff: noqa: F403,F405
from runtime_test_support import *


def test_stream_generate_question_uses_agent_loop_before_fallback() -> None:
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
                            "id": "call_context",
                            "function": {
                                "name": "recall_training_context",
                                "arguments": "{}",
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"question":"请讲讲 Redis 一致性。",'
                        '"expected_points":["目标","取舍","失败兜底","适用边界"]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    events = list(
        runtime.stream_generate_question(
            GenerateQuestionRequest(mode="basics", topic="redis", intensity="standard")
        )
    )

    context_events = [event for event in events if event["type"] == "context"]
    result_event = events[-1]
    assert context_events[0]["name"] == "recall_training_context"
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["question"] == "请讲讲 Redis 一致性。"
    assert '"question":"请讲讲 Redis 一致性。"' in result_event["data"]["raw_output"]


def test_stream_evaluate_answer_uses_agent_loop_and_emits_side_effects() -> None:
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
                            "id": "call_context",
                            "function": {
                                "name": "recall_training_context",
                                "arguments": "{}",
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content="",
                    tool_calls=[
                        {
                            "id": "call_observation",
                            "function": {
                                "name": "record_observation",
                                "arguments": (
                                    '{"category":"pattern","content":"回答主线清楚，但案例不够具体。",'
                                    '"tags":["redis"],"topic":"redis","relevance":0.9}'
                                ),
                            },
                        },
                        {
                            "id": "call_depth",
                            "function": {
                                "name": "set_depth_signal",
                                "arguments": '{"depth_signal":"skip_followup"}',
                            },
                        },
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"score":88,"score_breakdown":{"准确性":88},"headline":"主线到位",'
                        '"strengths":["主线清楚"],"gaps":["案例不够具体"],'
                        '"suggestion":"补真实案例","followup_question":"",'
                        '"followup_expected_points":[],"weakness_hits":[]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    events = list(
        runtime.stream_evaluate_answer(
            EvaluateAnswerRequest(
                mode="basics",
                topic="redis",
                question="Redis 为什么快？",
                expected_points=["内存访问", "事件循环"],
                answer="因为它主要在内存里。",
                turn_index=1,
                max_turns=2,
                agent_context={
                    "observations": [
                        {
                            "id": "obs_trace_1",
                            "category": "pattern",
                            "content": "用户主线清楚，但 trade-off 不够展开。" * 8,
                            "tags": ["redis", "tradeoff", "detail", "ops", "extra"],
                            "topic": "redis",
                            "scope_type": "global",
                            "scope_id": "",
                            "relevance": 0.8,
                        }
                    ]
                },
            )
        )
    )

    context_names = [event["name"] for event in events if event["type"] == "context"]
    trace_codes = [event["data"]["code"] for event in events if event["type"] == "trace"]
    result_event = events[-1]
    assert "recall_training_context" in context_names
    assert "record_observation" in context_names
    assert "set_depth_signal" in context_names
    assert "runtime_started" in trace_codes
    assert "context_compacted" in trace_codes
    assert "runtime_completed" in trace_codes
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["score"] == 88
    assert result_event["data"]["side_effects"]["depth_signal"] == "skip_followup"
    assert len(result_event["data"]["side_effects"]["observations"]) == 1
    compacted = next(
        event["data"]
        for event in events
        if event["type"] == "trace" and event["data"]["code"] == "context_compacted"
    )
    assert compacted["details"]["section"] == "observations"
    assert compacted["details"]["after_chars"] < compacted["details"]["before_chars"]
    assert result_event["data"]["trace"]["entries"][-1]["code"] == "runtime_completed"


def test_stream_generate_review_uses_agent_loop_and_emits_recommended_next_side_effect() -> None:
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
                            "id": "call_context",
                            "function": {
                                "name": "recall_training_context",
                                "arguments": "{}",
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content="",
                    tool_calls=[
                        {
                            "id": "call_next",
                            "function": {
                                "name": "suggest_next_session",
                                "arguments": (
                                    '{"mode":"basics","topic":"redis",'
                                    '"reason":"继续补缓存一致性表达"}'
                                ),
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"overall":"总结","top_fix":"先补缓存一致性取舍",'
                        '"top_fix_reason":"这是这轮最缺说服力的地方",'
                        '"highlights":["主线清楚"],"gaps":["案例偏少"],'
                        '"suggested_topics":["redis"],"next_training_focus":["补细节"],'
                        '"score_breakdown":{"准确性":80}}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    events = list(
        runtime.stream_generate_review(
            GenerateReviewRequest(
                session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
                turns=[
                    TrainingTurn(
                        question="Redis 为什么快？",
                        expected_points=["内存访问", "事件循环"],
                        answer="因为它主要在内存里。",
                    )
                ],
            )
        )
    )

    context_names = [event["name"] for event in events if event["type"] == "context"]
    result_event = events[-1]
    assert "recall_training_context" in context_names
    assert "suggest_next_session" in context_names
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["top_fix"] == "先补缓存一致性取舍"
    assert result_event["data"]["side_effects"]["recommended_next"]["topic"] == "redis"


def test_stream_generate_question_result_event_wraps_raw_output() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeStreamModelClient([]),
    )

    events = list(
        runtime.stream_generate_question(
            GenerateQuestionRequest(mode="basics", topic="redis", intensity="standard")
        )
    )

    result_event = events[-1]
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["question"] == "请讲讲 Redis 一致性。"
    assert '"question":"请讲讲 Redis 一致性。"' in result_event["data"]["raw_output"]


def test_stream_evaluate_answer_result_event_wraps_raw_output() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeEvaluateStreamModelClient([]),
    )

    events = list(
        runtime.stream_evaluate_answer(
            EvaluateAnswerRequest(
                mode="basics",
                topic="redis",
                question="Redis 为什么快？",
                expected_points=["内存访问", "事件循环"],
                answer="因为它在内存里。",
                turn_index=1,
                max_turns=1,
            )
        )
    )

    result_event = events[-1]
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["score"] == 86
    assert '"score":86' in result_event["data"]["raw_output"]


def test_stream_generate_review_result_event_wraps_raw_output() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeReviewStreamModelClient([]),
    )

    events = list(
        runtime.stream_generate_review(
            GenerateReviewRequest(
                session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
                turns=[
                    TrainingTurn(
                        question="Redis 为什么快？",
                        expected_points=["内存访问", "事件循环"],
                        answer="因为它在内存里。",
                    )
                ],
            )
        )
    )

    result_event = events[-1]
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["top_fix"] == "先补关键缺口"
    assert '"top_fix":"先补关键缺口"' in result_event["data"]["raw_output"]
