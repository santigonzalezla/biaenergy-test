from app.analysis.pipeline import Explainer
from app.config import Settings
from app.explanation.llm import LlmExplainer
from app.explanation.template import TemplateExplainer


def build_explainer(settings: Settings) -> Explainer:
    templates = TemplateExplainer(settings.timezone)
    if not settings.llm_enabled:
        return templates

    return LlmExplainer.from_api_key(
        api_key=settings.llm_api_key,
        model=settings.llm_model,
        timezone=settings.timezone,
        fallback=templates,
    )
