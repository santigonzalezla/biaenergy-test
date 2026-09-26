from app.analysis.pipeline import Explainer
from app.config import Settings
from app.explanation.template import TemplateExplainer


def build_explainer(settings: Settings) -> Explainer:
    return TemplateExplainer(settings.timezone)
