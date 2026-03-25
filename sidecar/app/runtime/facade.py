from __future__ import annotations

import logging
import time
from collections.abc import Iterator
from typing import Any

from app.adapters.go_client import GoBackendClient
from app.adapters.llm_client import ModelClientError, OpenAICompatibleModelClient
from app.config import Settings
from app.repo_analysis.context import RepoAnalysisBundle, collect_repo_analysis_bundle
from app.runtime.engine import run_task
from app.runtime.specs import (
    build_analyze_job_target_spec,
    build_analyze_repo_spec,
    build_evaluate_answer_spec,
    build_generate_question_spec,
    build_generate_review_spec,
)
from app.runtime.state import TaskExecutionResult
from app.runtime.streaming import stream_task
from app.schemas import (
    AnalyzeJobTargetRequest,
    AnalyzeJobTargetResponse,
    AnalyzeRepoRequest,
    AnalyzeRepoResponse,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewRequest,
    ReviewCard,
)

logger = logging.getLogger(__name__)


class AgentRuntime:
    # facade 层只负责把请求翻译成 task spec，再委托给通用 runtime。
    # 这样每个任务的 prompt/tool 组合可以演进，但执行状态机始终只有一套。
    def __init__(
        self,
        settings: Settings,
        *,
        model_client: OpenAICompatibleModelClient | None = None,
        go_client: GoBackendClient | None = None,
    ) -> None:
        self._settings = settings
        self._model_client = model_client
        self._go_client = go_client or GoBackendClient(settings)
        if self._model_client is None and settings.llm_enabled:
            self._model_client = OpenAICompatibleModelClient(settings)

    def collect_repo_bundle(self, request: AnalyzeRepoRequest) -> RepoAnalysisBundle:
        self._require_model_client()
        return collect_repo_analysis_bundle(request, self._settings)

    def summarize_repo_bundle(
        self,
        bundle: RepoAnalysisBundle,
    ) -> TaskExecutionResult[AnalyzeRepoResponse]:
        started_at = time.perf_counter()
        logger.info("analyze_repo summarize started repo_url=%s", bundle.repo_url)
        spec = build_analyze_repo_spec(bundle)
        draft_result = run_task(
            model_client=self._require_model_client(),
            task_name="analyze_repo",
            response_model=spec.response_model,
            system_prompt=spec.system_prompt,
            user_prompt=spec.user_prompt,
            tools=spec.tools,
            fallback_tools=spec.fallback_tools,
            result_validator=spec.result_validator,
            context_trace_details=spec.context_trace_details,
        )
        response = AnalyzeRepoResponse(
            repo_url=bundle.repo_url,
            name=bundle.name,
            default_branch=bundle.default_branch,
            import_commit=bundle.import_commit,
            summary=draft_result.result.summary,
            tech_stack=bundle.tech_stack,
            highlights=draft_result.result.highlights,
            challenges=draft_result.result.challenges,
            tradeoffs=draft_result.result.tradeoffs,
            ownership_points=draft_result.result.ownership_points,
            followup_points=draft_result.result.followup_points,
            chunks=bundle.chunks,
        )
        logger.info(
            "analyze_repo summarize completed repo_url=%s duration_ms=%.2f",
            bundle.repo_url,
            (time.perf_counter() - started_at) * 1000,
        )
        return TaskExecutionResult(
            result=response,
            raw_output=draft_result.raw_output,
            trace=draft_result.trace,
        )

    def analyze_repo(self, request: AnalyzeRepoRequest) -> AnalyzeRepoResponse:
        bundle = self.collect_repo_bundle(request)
        return self.summarize_repo_bundle(bundle).result

    def analyze_job_target_task(
        self,
        request: AnalyzeJobTargetRequest,
    ) -> TaskExecutionResult[AnalyzeJobTargetResponse]:
        started_at = time.perf_counter()
        logger.info("analyze_job_target started title=%s", request.title)
        spec = build_analyze_job_target_spec(request)
        draft_result = run_task(
            model_client=self._require_model_client(),
            task_name="analyze_job_target",
            response_model=spec.response_model,
            system_prompt=spec.system_prompt,
            user_prompt=spec.user_prompt,
            tools=spec.tools,
            fallback_tools=spec.fallback_tools,
            result_validator=spec.result_validator,
            context_trace_details=spec.context_trace_details,
        )
        response = AnalyzeJobTargetResponse(
            summary=draft_result.result.summary,
            must_have_skills=draft_result.result.must_have_skills,
            bonus_skills=draft_result.result.bonus_skills,
            responsibilities=draft_result.result.responsibilities,
            evaluation_focus=draft_result.result.evaluation_focus,
        )
        logger.info(
            "analyze_job_target completed title=%s duration_ms=%.2f",
            request.title,
            (time.perf_counter() - started_at) * 1000,
        )
        return TaskExecutionResult(
            result=response,
            raw_output=draft_result.raw_output,
            trace=draft_result.trace,
        )

    def analyze_job_target(self, request: AnalyzeJobTargetRequest) -> AnalyzeJobTargetResponse:
        return self.analyze_job_target_task(request).result

    def generate_question_task(
        self,
        request: GenerateQuestionRequest,
    ) -> TaskExecutionResult[GenerateQuestionResponse]:
        started_at = time.perf_counter()
        logger.info("generate_question started mode=%s topic=%s", request.mode, request.topic)
        spec = build_generate_question_spec(request, self._go_client)
        response = run_task(
            model_client=self._require_model_client(),
            task_name="generate_question",
            response_model=spec.response_model,
            system_prompt=spec.system_prompt,
            user_prompt=spec.user_prompt,
            tools=spec.tools,
            fallback_tools=spec.fallback_tools,
            result_validator=spec.result_validator,
            context_trace_details=spec.context_trace_details,
        )
        logger.info(
            "generate_question completed mode=%s topic=%s duration_ms=%.2f",
            request.mode,
            request.topic,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def generate_question(self, request: GenerateQuestionRequest) -> GenerateQuestionResponse:
        return self.generate_question_task(request).result

    def stream_generate_question(
        self, request: GenerateQuestionRequest
    ) -> Iterator[dict[str, Any]]:
        spec = build_generate_question_spec(request, self._go_client)
        yield from stream_task(
            model_client=self._require_model_client(),
            task_name="generate_question",
            response_model=spec.response_model,
            system_prompt=spec.system_prompt,
            user_prompt=spec.user_prompt,
            tools=spec.tools,
            fallback_tools=spec.fallback_tools,
            result_validator=spec.result_validator,
            context_trace_details=spec.context_trace_details,
        )

    def evaluate_answer_task(
        self,
        request: EvaluateAnswerRequest,
    ) -> TaskExecutionResult[EvaluationResult]:
        started_at = time.perf_counter()
        logger.info(
            "evaluate_answer started mode=%s topic=%s turn=%d/%d",
            request.mode,
            request.topic,
            request.turn_index,
            request.max_turns,
        )
        spec = build_evaluate_answer_spec(request, self._go_client)
        response = run_task(
            model_client=self._require_model_client(),
            task_name="evaluate_answer",
            response_model=spec.response_model,
            system_prompt=spec.system_prompt,
            user_prompt=spec.user_prompt,
            tools=spec.tools,
            fallback_tools=spec.fallback_tools,
            result_validator=spec.result_validator,
            context_trace_details=spec.context_trace_details,
        )
        logger.info(
            "evaluate_answer completed mode=%s topic=%s turn=%d/%d duration_ms=%.2f",
            request.mode,
            request.topic,
            request.turn_index,
            request.max_turns,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def evaluate_answer(self, request: EvaluateAnswerRequest) -> EvaluationResult:
        return self.evaluate_answer_task(request).result

    def stream_evaluate_answer(self, request: EvaluateAnswerRequest) -> Iterator[dict[str, Any]]:
        spec = build_evaluate_answer_spec(request, self._go_client)
        yield from stream_task(
            model_client=self._require_model_client(),
            task_name="evaluate_answer",
            response_model=spec.response_model,
            system_prompt=spec.system_prompt,
            user_prompt=spec.user_prompt,
            tools=spec.tools,
            fallback_tools=spec.fallback_tools,
            result_validator=spec.result_validator,
            context_trace_details=spec.context_trace_details,
        )

    def generate_review_task(
        self,
        request: GenerateReviewRequest,
    ) -> TaskExecutionResult[ReviewCard]:
        started_at = time.perf_counter()
        logger.info("generate_review started session_id=%s", request.session.id)
        spec = build_generate_review_spec(request, self._go_client)
        response = run_task(
            model_client=self._require_model_client(),
            task_name="generate_review",
            response_model=spec.response_model,
            system_prompt=spec.system_prompt,
            user_prompt=spec.user_prompt,
            tools=spec.tools,
            fallback_tools=spec.fallback_tools,
            result_validator=spec.result_validator,
            context_trace_details=spec.context_trace_details,
        )
        logger.info(
            "generate_review completed session_id=%s duration_ms=%.2f",
            request.session.id,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def generate_review(self, request: GenerateReviewRequest) -> ReviewCard:
        return self.generate_review_task(request).result

    def stream_generate_review(self, request: GenerateReviewRequest) -> Iterator[dict[str, Any]]:
        spec = build_generate_review_spec(request, self._go_client)
        yield from stream_task(
            model_client=self._require_model_client(),
            task_name="generate_review",
            response_model=spec.response_model,
            system_prompt=spec.system_prompt,
            user_prompt=spec.user_prompt,
            tools=spec.tools,
            fallback_tools=spec.fallback_tools,
            result_validator=spec.result_validator,
            context_trace_details=spec.context_trace_details,
        )

    def _require_model_client(self) -> OpenAICompatibleModelClient:
        if self._model_client is None:
            raise ModelClientError(
                "LLM is required for sidecar core flows. Configure PRACTICEHELPER_SIDECAR_MODEL "
                "and PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL. If the provider requires auth, also "
                "set PRACTICEHELPER_SIDECAR_OPENAI_API_KEY."
            )
        return self._model_client
