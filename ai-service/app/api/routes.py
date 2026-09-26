from fastapi import APIRouter

from app.analysis.pipeline import AnomalyAnalyzer
from app.domain.models import AnalysisRequest, AnalysisResult


def build_router(analyzer: AnomalyAnalyzer, explainer_name: str) -> APIRouter:
    router = APIRouter()

    @router.get("/health")
    def health() -> dict[str, str]:
        return {"status": "ok", "explainer": explainer_name}

    @router.post("/analyze", response_model=AnalysisResult)
    def analyze(request: AnalysisRequest) -> AnalysisResult:
        return analyzer.analyze(request)

    return router
