import logging

from fastapi import FastAPI

from app.analysis.pipeline import AnomalyAnalyzer
from app.api.errors import register_error_handlers
from app.api.routes import build_router
from app.config import Settings, load_settings
from app.explanation.factory import build_explainer

logger = logging.getLogger(__name__)


def create_app(settings: Settings | None = None) -> FastAPI:
    settings = settings or load_settings()

    explainer = build_explainer(settings)
    explainer_name = type(explainer).__name__
    analyzer = AnomalyAnalyzer(explainer)

    app = FastAPI(title="BiaEnergy AI Service", version="0.1.0")
    register_error_handlers(app)
    app.include_router(build_router(analyzer, explainer_name))

    logger.info("ai-service ready (environment=%s, explainer=%s)", settings.environment, explainer_name)
    return app
