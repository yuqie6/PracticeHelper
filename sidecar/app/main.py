from __future__ import annotations

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from app.config import load_settings
from app.langgraph_flows import build_flows
from app.llm_client import ModelClientError
from app.schemas import (
    AnalyzeRepoRequest,
    AnalyzeRepoResponse,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewRequest,
    ReviewCard,
)

settings = load_settings()
flows = build_flows(settings)

app = FastAPI(title="PracticeHelper Sidecar", version="0.1.0")


@app.exception_handler(ModelClientError)
def handle_model_client_error(_: Request, exc: ModelClientError) -> JSONResponse:
    message = str(exc)
    status_code = 503 if "LLM is required" in message else 502
    return JSONResponse(status_code=status_code, content={"error": {"message": message}})


@app.get("/healthz")
def healthcheck() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/internal/analyze_repo", response_model=AnalyzeRepoResponse)
def analyze_repo_endpoint(request: AnalyzeRepoRequest) -> AnalyzeRepoResponse:
    return flows["analyze_repo"].invoke({"request": request})["result"]


@app.post("/internal/generate_question", response_model=GenerateQuestionResponse)
def generate_question_endpoint(request: GenerateQuestionRequest) -> GenerateQuestionResponse:
    return flows["generate_question"].invoke({"request": request})["result"]


@app.post("/internal/evaluate_answer", response_model=EvaluationResult)
def evaluate_answer_endpoint(request: EvaluateAnswerRequest) -> EvaluationResult:
    return flows["evaluate_answer"].invoke({"request": request})["result"]


@app.post("/internal/generate_review", response_model=ReviewCard)
def generate_review_endpoint(request: GenerateReviewRequest) -> ReviewCard:
    return flows["generate_review"].invoke({"request": request})["result"]
