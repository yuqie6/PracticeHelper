# ruff: noqa: F403,F405
from runtime_test_support import *


def test_evaluate_answer_task_accepts_transition_session_command_result() -> None:
    backend_client = FakeBackendClient()
    backend_client.command_results["transition_session"] = AgentCommandResult(
        command_id="cmd_transition_session_turn_1_skip_followup",
        status="deferred",
        data={
            "resolved_depth_signal": "skip_followup",
            "resolved_max_turns": 2,
        },
    )
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
                            "id": "call_transition",
                            "function": {
                                "name": "transition_session",
                                "arguments": (
                                    '{"decision":"skip_followup","reason":"本轮已经看清主线"}'
                                ),
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"score":90,"score_breakdown":{"准确性":90},"headline":"可以直接收口",'
                        '"strengths":["主线清楚"],"gaps":["还可以补更多案例"],"suggestion":"继续保持",'
                        '"followup_question":"","followup_expected_points":[],"weakness_hits":[]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
        go_client=backend_client,
    )

    result = runtime.evaluate_answer_task(
        EvaluateAnswerRequest(
            session_id="sess_cmd_eval",
            mode="basics",
            topic="redis",
            question="Redis 为什么快？",
            expected_points=["内存访问", "事件循环"],
            answer="因为主要在内存里，事件循环模型也简单。",
            turn_index=1,
            max_turns=2,
        )
    )

    assert result.command_results[0]["status"] == "deferred"
    assert result.result.followup_question == ""
    assert backend_client.commands[0].session_id == "sess_cmd_eval"
    assert any(entry.phase == "command" for entry in result.trace.entries)


def test_transition_session_tool_dedupes_same_command_within_single_run() -> None:
    backend_client = FakeBackendClient()
    backend_client.command_results["transition_session"] = AgentCommandResult(
        command_id="cmd_transition_session_turn_1_extend",
        status="deferred",
        data={
            "resolved_depth_signal": "extend",
            "resolved_max_turns": 3,
        },
    )
    tool = build_evaluate_answer_agent_tools(
        EvaluateAnswerRequest(
            session_id="sess_cmd_dedupe",
            mode="basics",
            topic="redis",
            question="Redis 为什么快？",
            expected_points=["内存访问"],
            answer="因为在内存里。",
            turn_index=1,
            max_turns=2,
        ),
        {},
        backend_client=backend_client,
    )
    by_name = {item.name: item for item in tool}
    state = type(
        "State",
        (),
        {
            "side_effects": {},
            "command_results": [],
            "command_cache": {},
            "command_counts": {},
            "command_budget": {"transition_session": 1, "upsert_review_path": 1},
        },
    )()

    bound = by_name["transition_session"].runtime_bind(state)
    first = bound.handler({"decision": "extend", "reason": "还得再追一刀"})
    second = bound.handler({"decision": "extend", "reason": "还得再追一刀"})

    assert first["status"] == "deferred"
    assert second["deduped"] is True
    assert len(backend_client.commands) == 1
    assert len(state.command_results) == 1


def test_generate_review_tools_include_session_detail_callback_when_backend_enabled() -> None:
    backend_client = FakeBackendClient()
    tools = build_generate_review_agent_tools(
        GenerateReviewRequest(
            session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
            turns=[],
        ),
        {},
        backend_client=backend_client,
    )

    by_name = {tool.name: tool for tool in tools}
    assert "get_session_detail" in by_name
    payload = by_name["get_session_detail"].handler({})
    assert payload["session"]["id"] == "sess_1"
    assert backend_client.session_ids == ["sess_1"]


def test_stream_generate_review_emits_command_status_and_command_results() -> None:
    backend_client = FakeBackendClient()
    backend_client.command_results["upsert_review_path"] = AgentCommandResult(
        command_id="cmd_upsert_review_path",
        status="applied",
        applied=True,
        data={
            "recommended_next": {
                "mode": "basics",
                "topic": "redis",
                "reason": "先补缓存一致性取舍",
            },
            "suggested_topics": ["redis"],
            "next_training_focus": ["补缓存一致性表达"],
        },
    )
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
                            "id": "call_path",
                            "function": {
                                "name": "upsert_review_path",
                                "arguments": (
                                    '{"recommended_next":{"mode":"basics","topic":"redis"},'
                                    '"suggested_topics":["redis"],'
                                    '"next_training_focus":["补缓存一致性表达"],'
                                    '"gaps":["缺缓存一致性取舍"],'
                                    '"top_fix":"补缓存一致性取舍",'
                                    '"top_fix_reason":"这是最影响训练效果的短板"}'
                                ),
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"overall":"总结","top_fix":"补缓存一致性取舍",'
                        '"top_fix_reason":"这是最影响训练效果的短板",'
                        '"highlights":["主线清楚"],"gaps":["缺缓存一致性取舍"],'
                        '"suggested_topics":["redis"],'
                        '"next_training_focus":["补缓存一致性表达"],'
                        '"recommended_next":{"mode":"basics","topic":"redis","reason":"先补缓存一致性取舍"},'
                        '"score_breakdown":{"准确性":72}}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
        go_client=backend_client,
    )

    events = list(
        runtime.stream_generate_review(
            GenerateReviewRequest(
                session=TrainingSession(id="sess_stream_review", mode="basics", topic="redis"),
                turns=[],
            )
        )
    )

    status_names = [event["name"] for event in events if event["type"] == "status"]
    assert "command_requested" in status_names
    assert "command_applied" in status_names

    result_event = next(event for event in events if event["type"] == "result")
    assert result_event["data"]["command_results"][0]["status"] == "applied"
    assert (
        result_event["data"]["command_results"][0]["data"]["recommended_next"]["topic"] == "redis"
    )
    assert any(event["type"] == "trace" and event["data"]["phase"] == "command" for event in events)
