import secrets

from fastapi import APIRouter, Depends, HTTPException, Security
from fastapi.security import APIKeyHeader

from app.analysis.pipeline import AnomalyAnalyzer
from app.domain.models import AnalysisRequest, AnalysisResult

SERVICE_TOKEN_HEADER = "X-Service-Token"

service_token_header = APIKeyHeader(name=SERVICE_TOKEN_HEADER, auto_error=False)


def build_router(analyzer: AnomalyAnalyzer, explainer_name: str, service_token: str | None = None) -> APIRouter:
    router = APIRouter()

    def require_service_token(provided: str | None = Security(service_token_header)) -> None:
        if service_token is None:
            return
        if provided is None or not secrets.compare_digest(provided, service_token):
            raise HTTPException(status_code=401, detail="Missing or invalid service token")

    @router.get("/health")
    def health() -> dict[str, str]:
        return {"status": "ok", "explainer": explainer_name}

    @router.post("/analyze", response_model=AnalysisResult, dependencies=[Depends(require_service_token)])
    def analyze(request: AnalysisRequest) -> AnalysisResult:
        return analyzer.analyze(request)

    return router
