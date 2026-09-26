import logging
from typing import Any

import anthropic
from pydantic import ValidationError

from app.analysis.pipeline import Explainer
from app.domain.models import Explanation, Finding

logger = logging.getLogger(__name__)

MAX_TOKENS = 4000
REQUEST_TIMEOUT_SECONDS = 60.0
SERVER_FALLBACK_BETA = "server-side-fallback-2026-07-01"
SERVER_FALLBACK_MODELS = ("claude-opus-5", "claude-fable-5")

SYSTEM_PROMPT = """\
Eres el analista de un sistema de gestión energética industrial. Recibes un hallazgo ya clasificado \
por un motor de reglas determinista, con toda su evidencia en JSON, y redactas dos textos para el \
operador de planta, que no conoce los detalles internos del sistema:

- reason: qué ocurrió, desde cuándo, con qué magnitud y por qué es relevante. Máximo 3 oraciones.
- recommendedAction: una sola acción concreta. Máximo 2 oraciones.

Cifras:
- Para el consumo usa únicamente variationPct, de baselineKwh a currentKwh (kWh por hora). No cites otro \
porcentaje de consumo.
- Menciona solo las variables eléctricas que cambiaron (factor de potencia, corriente, voltaje) con su valor \
de referencia y el observado, redondeados.

La acción depende del tipo de hallazgo:
- REAL_ANOMALY: revisar los equipos y cargas asociados al medidor (motores, compresores, banco de \
condensadores) y confirmar con operación si hubo un cambio no registrado. El problema está en la \
instalación, no en el medidor.
- DATA_QUALITY: revisar el propio medidor y su instrumentación, y no confiar en sus lecturas hasta verificarlo.
- EXPLAINABLE_ANOMALY: validar con operación que el nuevo nivel es el esperado y actualizar la línea base.
- FALSE_POSITIVE: indicar que no requiere acción correctiva.

Reglas:
- La clasificación ya está decidida: explícala, no la cuestiones ni la cambies.
- No menciones identificadores internos (ruleId, nombres de reglas o de campos), el puntaje de confianza \
ni el JSON.
- Usa solo hechos presentes en la evidencia. No inventes cifras ni causas; en la acción puedes proponer \
qué revisar, pero no afirmes ni sugieras causas que la evidencia no respalda.
- Un evento de tipo UNKNOWN significa que la planta confirmó que no hay una causa operativa conocida.
- Escribe en español, con coma decimal y las fechas en la zona horaria indicada (por ejemplo, \
"12 de septiembre a las 14:00")."""


class LlmExplainer:
    def __init__(self, client: Any, model: str, timezone: str, fallback: Explainer):
        self._client = client
        self._model = model
        self._timezone = timezone
        self._fallback = fallback

    @classmethod
    def from_api_key(cls, api_key: str, model: str, timezone: str, fallback: Explainer) -> "LlmExplainer":
        client = anthropic.Anthropic(api_key=api_key, timeout=REQUEST_TIMEOUT_SECONDS, max_retries=2)
        return cls(client=client, model=model, timezone=timezone, fallback=fallback)

    def explain(self, finding: Finding) -> Explanation:
        try:
            return self._ask_model(finding)
        except (anthropic.APIError, ValidationError, ExplanationUnavailable) as error:
            logger.warning(
                "llm explanation failed, using fallback (meter=%s, rule=%s): %s",
                finding.meter_code,
                finding.rule_id,
                error,
            )
            return self._fallback.explain(finding)

    def _ask_model(self, finding: Finding) -> Explanation:
        response = self._client.beta.messages.parse(
            model=self._model,
            max_tokens=MAX_TOKENS,
            system=SYSTEM_PROMPT,
            messages=[{"role": "user", "content": self._prompt(finding)}],
            output_format=Explanation,
            **self._server_fallback(),
        )

        if response.stop_reason == "refusal":
            raise ExplanationUnavailable("the model declined the request")

        explanation = response.parsed_output
        if explanation is None or not explanation.reason.strip() or not explanation.recommended_action.strip():
            raise ExplanationUnavailable("the model returned an empty explanation")

        return explanation

    def _prompt(self, finding: Finding) -> str:
        evidence = finding.model_dump_json(by_alias=True, exclude={"reason", "recommended_action"}, indent=2)
        return f"Zona horaria del sitio: {self._timezone}\n\nHallazgo:\n{evidence}"

    def _server_fallback(self) -> dict[str, Any]:
        if not self._model.startswith(SERVER_FALLBACK_MODELS):
            return {}
        return {"betas": [SERVER_FALLBACK_BETA], "fallbacks": "default"}


class ExplanationUnavailable(Exception):
    pass
