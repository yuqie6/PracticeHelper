# ruff: noqa: F403,F405
from runtime_test_support import *


def test_evaluate_answer_task_retries_inside_agent_loop_after_validation_failure() -> None:
    client = FakeModelClient(
        [
            ChatCompletionResult(
                content="",
                tool_calls=[
                    {
                        "id": "call_ctx",
                        "function": {
                            "name": "recall_training_context",
                            "arguments": "{}",
                        },
                    }
                ],
            ),
            ChatCompletionResult(
                content='{"score":80,"score_breakdown":{"准确性":80}}',
                tool_calls=[],
            ),
            ChatCompletionResult(
                content=(
                    '{"score":82,"score_breakdown":{"准确性":82},"strengths":["主线清楚"],'
                    '"gaps":["例子不够具体"],"followup_question":"如果线上抖动，你会先看什么？",'
                    '"followup_expected_points":["先止血","再定位"],"weakness_hits":[]}'
                ),
                tool_calls=[],
            ),
        ]
    )
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=client,
    )

    result = runtime.evaluate_answer_task(
        EvaluateAnswerRequest(
            mode="basics",
            topic="redis",
            question="Redis 为什么快？",
            expected_points=["内存访问", "事件循环"],
            answer="因为它在内存里。",
            turn_index=1,
            max_turns=2,
        )
    )

    assert result.result.followup_question == "如果线上抖动，你会先看什么？"
    assert len(client.calls) == 3
    assert any(entry.code == "semantic_validation_failed" for entry in result.trace.entries)
    assert any(
        message["role"] == "user" and "missing strengths/gaps" in message["content"]
        for message in client.calls[-1]["messages"]
    )


def test_evaluate_answer_task_accepts_skip_followup_signal_without_followup_output() -> None:
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
                            "id": "call_ctx",
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
                            "id": "call_depth",
                            "function": {
                                "name": "set_depth_signal",
                                "arguments": '{"depth_signal":"skip_followup"}',
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"score":91,"score_breakdown":{"准确性":91},"headline":"可以直接收口",'
                        '"strengths":["主线清楚"],"gaps":["还可以补更多案例"],"suggestion":"继续保持",'
                        '"followup_question":"","followup_expected_points":[],"weakness_hits":[]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    result = runtime.evaluate_answer_task(
        EvaluateAnswerRequest(
            mode="basics",
            topic="redis",
            question="Redis 为什么快？",
            expected_points=["内存访问", "事件循环"],
            answer="因为主要在内存里，事件循环模型也简单。",
            turn_index=1,
            max_turns=2,
        )
    )

    assert result.side_effects["depth_signal"] == "skip_followup"
    assert result.result.followup_question == ""


def test_evaluate_answer_task_accepts_extend_signal_with_followup_on_last_turn() -> None:
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
                            "id": "call_ctx",
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
                            "id": "call_depth",
                            "function": {
                                "name": "set_depth_signal",
                                "arguments": '{"depth_signal":"extend"}',
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"score":66,"score_breakdown":{"准确性":66},"headline":"还需要继续追问",'
                        '"strengths":["主线大体对"],"gaps":["止血顺序不完整"],"suggestion":"补具体排查顺序",'
                        '"followup_question":"如果线上抖动，你先止血还是先定位？为什么？",'
                        '"followup_expected_points":["先止血","再定位","说明取舍"],"weakness_hits":[]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    result = runtime.evaluate_answer_task(
        EvaluateAnswerRequest(
            mode="basics",
            topic="redis",
            question="Redis 为什么快？",
            expected_points=["内存访问", "事件循环"],
            answer="因为主要在内存里。",
            turn_index=2,
            max_turns=2,
        )
    )

    assert result.side_effects["depth_signal"] == "extend"
    assert "如果线上抖动" in result.result.followup_question


def test_evaluate_answer_task_raises_after_agent_loop_validation_budget_is_exhausted() -> None:
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
                            "id": "call_ctx",
                            "function": {
                                "name": "recall_training_context",
                                "arguments": "{}",
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content='{"score":80,"score_breakdown":{"准确性":80}}',
                    tool_calls=[],
                ),
                ChatCompletionResult(
                    content='{"score":81,"score_breakdown":{"准确性":81}}',
                    tool_calls=[],
                ),
                ChatCompletionResult(
                    content='{"score":82,"score_breakdown":{"准确性":82}}',
                    tool_calls=[],
                ),
            ]
        ),
    )

    with pytest.raises(ModelClientError, match="missing strengths/gaps"):
        runtime.evaluate_answer_task(
            EvaluateAnswerRequest(
                mode="basics",
                topic="redis",
                question="Redis 为什么快？",
                expected_points=["内存访问", "事件循环"],
                answer="因为它在内存里。",
                turn_index=1,
                max_turns=2,
            )
        )


def test_generate_review_task_retries_inside_agent_loop_after_validation_failure() -> None:
    client = FakeModelClient(
        [
            ChatCompletionResult(
                content="",
                tool_calls=[
                    {
                        "id": "call_ctx",
                        "function": {
                            "name": "recall_training_context",
                            "arguments": "{}",
                        },
                    }
                ],
            ),
            ChatCompletionResult(
                content='{"overall":"总结","score_breakdown":{"准确性":70}}',
                tool_calls=[],
            ),
            ChatCompletionResult(
                content=(
                    '{"overall":"总结","top_fix":"先补最关键缺口","top_fix_reason":"这是当前最影响说服力的部分",'
                    '"highlights":["主线清楚"],"gaps":["案例不够具体"],"suggested_topics":["redis"],'
                    '"next_training_focus":["补细节"],"recommended_next":{"mode":"basics","topic":"redis","reason":"补短板"},'
                    '"score_breakdown":{"准确性":70}}'
                ),
                tool_calls=[],
            ),
        ]
    )
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=client,
    )

    result = runtime.generate_review_task(
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

    assert result.result.top_fix == "先补最关键缺口"
    assert len(client.calls) == 3
    assert any(
        message["role"] == "user" and "missing top_fix" in message["content"]
        for message in client.calls[-1]["messages"]
    )


def test_generate_review_task_accepts_recommended_next_from_side_effects_only() -> None:
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
                            "id": "call_ctx",
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
                                    '{"mode":"basics","topic":"redis","reason":"先补缓存击穿止血顺序"}'
                                ),
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"overall":"总结","top_fix":"先补最关键缺口","top_fix_reason":"这是当前最影响说服力的部分",'
                        '"highlights":["主线清楚"],"gaps":["案例不够具体"],"suggested_topics":["redis"],'
                        '"next_training_focus":["补细节"],"recommended_next":null,"score_breakdown":{"准确性":70}}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    result = runtime.generate_review_task(
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

    assert result.result.recommended_next is None
    assert result.side_effects["recommended_next"]["topic"] == "redis"


def test_evaluate_answer_returns_side_effects_from_action_tools() -> None:
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
                            "id": "call_ctx",
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
                            "id": "call_obs",
                            "function": {
                                "name": "record_observation",
                                "arguments": (
                                    '{"category":"pattern","content":"用户会讲主线，但案例不够具体。",'
                                    '"tags":["表达","案例"],"relevance":0.9,"topic":"redis"}'
                                ),
                            },
                        },
                        {
                            "id": "call_kn",
                            "function": {
                                "name": "update_knowledge",
                                "arguments": (
                                    '{"label":"redis","node_type":"topic","proficiency":2.5,'
                                    '"confidence":0.8,"evidence":"能讲清主线，但案例不足"}'
                                ),
                            },
                        },
                        {
                            "id": "call_depth",
                            "function": {
                                "name": "set_depth_signal",
                                "arguments": '{"depth_signal":"extend"}',
                            },
                        },
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"score":78,"score_breakdown":{"准确性":78},"headline":"主线清楚",'
                        '"strengths":["主线完整"],"gaps":["案例不够具体"],"suggestion":"补真实案例",'
                        '"followup_question":"如果线上抖动，你先看什么？",'
                        '"followup_expected_points":["先止血","再定位"],'
                        '"weakness_hits":[{"kind":"detail","label":"案例不够具体","severity":0.7}]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    graph = _build_evaluate_answer_graph(runtime)
    result = graph.invoke(
        {
            "request": EvaluateAnswerRequest(
                mode="basics",
                topic="redis",
                question="Redis 为什么快？",
                expected_points=["内存访问", "事件循环"],
                answer="因为主要在内存里，线程模型也比较简单。",
                turn_index=1,
                max_turns=2,
            )
        }
    )

    envelope = result["result"]
    assert isinstance(envelope, EvaluateAnswerEnvelope)
    assert envelope.side_effects.depth_signal == "extend"
    assert len(envelope.side_effects.observations) == 1
    assert envelope.side_effects.observations[0].content == "用户会讲主线，但案例不够具体。"
    assert len(envelope.side_effects.knowledge_updates) == 1
    assert envelope.side_effects.knowledge_updates[0].label == "redis"


def test_generate_review_returns_recommended_next_from_action_tool() -> None:
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
                            "id": "call_session",
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
                                    '{"mode":"basics","topic":"redis","reason":"先补缓存击穿止血顺序"}'
                                ),
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"overall":"总结","top_fix":"先补关键缺口","top_fix_reason":"这是最大短板",'
                        '"highlights":["主线清楚"],"gaps":["案例不够具体"],'
                        '"suggested_topics":["redis"],"next_training_focus":["补细节"],'
                        '"recommended_next":{"mode":"basics","topic":"redis","reason":"补短板"},'
                        '"score_breakdown":{"准确性":72}}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    graph = _build_generate_review_graph(runtime)
    result = graph.invoke(
        {
            "request": GenerateReviewRequest(
                session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
                turns=[
                    TrainingTurn(
                        question="Redis 为什么快？",
                        expected_points=["内存访问", "事件循环"],
                        answer="因为它主要在内存里。",
                    )
                ],
            )
        }
    )

    envelope = result["result"]
    assert isinstance(envelope, GenerateReviewEnvelope)
    assert envelope.side_effects.recommended_next is not None
    assert envelope.side_effects.recommended_next.topic == "redis"
    assert envelope.side_effects.recommended_next.reason == "先补缓存击穿止血顺序"
