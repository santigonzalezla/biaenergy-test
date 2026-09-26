from types import SimpleNamespace

import anthropic
import httpx2
import pytest

from app.analysis.pipeline import AnomalyAnalyzer
from app.config import load_settings
from app.domain.models import Explanation
from app.explanation.factory import build_explainer
from app.explanation.llm import LlmExplainer
from app.explanation.template import TemplateExplainer


class FakeMessages:
    def __init__(self, response=None, error=None):
        self.response = response
        self.error = error
        self.calls: list[dict] = []

    def parse(self, **kwargs):
        self.calls.append(kwargs)
        if self.error is not None:
            raise self.error
        return self.response


def fake_client(response=None, error=None):
    messages = FakeMessages(response=response, error=error)
    return SimpleNamespace(beta=SimpleNamespace(messages=messages)), messages


def answer(reason="El consumo subió.", action="Inspeccionar el equipo.", stop_reason="end_turn"):
    return SimpleNamespace(
        stop_reason=stop_reason,
        parsed_output=Explanation(reason=reason, recommended_action=action),
    )


def explainer(client, model="claude-sonnet-5"):
    return LlmExplainer(client=client, model=model, timezone="America/Bogota", fallback=TemplateExplainer())


@pytest.fixture(scope="module")
def m109(dataset_request):
    result = AnomalyAnalyzer(TemplateExplainer()).analyze(dataset_request)
    return result.findings[0]


def test_uses_the_model_explanation(m109):
    client, _ = fake_client(response=answer())

    explanation = explainer(client).explain(m109)

    assert explanation == Explanation(reason="El consumo subió.", recommended_action="Inspeccionar el equipo.")


def test_sends_the_evidence_but_not_previous_texts(m109):
    client, messages = fake_client(response=answer())

    explainer(client).explain(m109)

    [call] = messages.calls
    prompt = call["messages"][0]["content"]
    assert call["model"] == "claude-sonnet-5"
    assert call["output_format"] is Explanation
    assert "America/Bogota" in prompt
    assert '"ruleId": "R4_UNEXPLAINED_SURGE"' in prompt
    assert "No operational event reported" in prompt
    assert '"reason"' not in prompt


@pytest.mark.parametrize(
    ("model", "expects_server_fallback"),
    [
        pytest.param("claude-sonnet-5", False, id="economic model without server fallback"),
        pytest.param("claude-opus-5", True, id="opus uses the default server fallback"),
    ],
)
def test_server_fallback_only_for_supported_models(m109, model, expects_server_fallback):
    client, messages = fake_client(response=answer())

    explainer(client, model=model).explain(m109)

    [call] = messages.calls
    assert ("fallbacks" in call) is expects_server_fallback


def connection_error():
    return anthropic.APIConnectionError(request=httpx2.Request("POST", "https://api.anthropic.com/v1/messages"))


@pytest.mark.parametrize(
    ("response", "error"),
    [
        pytest.param(None, connection_error(), id="network failure"),
        pytest.param(answer(stop_reason="refusal"), None, id="model refusal"),
        pytest.param(answer(reason="   "), None, id="empty explanation"),
    ],
)
def test_falls_back_to_templates_when_the_model_cannot_answer(m109, response, error):
    client, _ = fake_client(response=response, error=error)

    explanation = explainer(client).explain(m109)

    assert explanation == TemplateExplainer().explain(m109)


def test_factory_uses_templates_without_a_key():
    assert isinstance(build_explainer(load_settings({})), TemplateExplainer)


def test_factory_uses_the_llm_with_a_key():
    assert isinstance(build_explainer(load_settings({"LLM_API_KEY": "test-key"})), LlmExplainer)
