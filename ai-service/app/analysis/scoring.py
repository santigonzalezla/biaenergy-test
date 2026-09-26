from dataclasses import dataclass

from app.analysis.rules import CONSUMPTION_KINDS, Classification
from app.domain.enums import AnomalyType, EventType, Severity

BASE_CONFIDENCE = 0.55
CORROBORATION_BONUS = 0.10
MAX_CORROBORATION_BONUS = 0.20
EVENT_CONTEXT_BONUS = 0.15
DURATION_MATCH_BONUS = 0.10
MAX_CONFIDENCE = 0.99

TYPE_WEIGHT: dict[AnomalyType, float] = {
    AnomalyType.REAL_ANOMALY: 1.0,
    AnomalyType.DATA_QUALITY: 0.7,
    AnomalyType.EXPLAINABLE_ANOMALY: 0.4,
    AnomalyType.FALSE_POSITIVE: 0.05,
}
SEVERITY_WEIGHT: dict[Severity, float] = {
    Severity.HIGH: 1.0,
    Severity.MEDIUM: 0.6,
    Severity.LOW: 0.3,
}
RESOLVED_WEIGHT = 0.6

EVENTS_SUPPORTING_TYPE: dict[AnomalyType, set[EventType]] = {
    AnomalyType.REAL_ANOMALY: {EventType.UNKNOWN},
    AnomalyType.DATA_QUALITY: {EventType.DATA_QUALITY},
}


@dataclass(frozen=True)
class Score:
    confidence: float
    priority: float


def score(classification: Classification) -> Score:
    confidence = _confidence(classification)
    priority = (
        100
        * TYPE_WEIGHT[classification.rule.anomaly_type]
        * SEVERITY_WEIGHT[classification.severity]
        * _impact(classification)
        * confidence
        * _activity(classification)
    )
    return Score(confidence=round(confidence, 2), priority=round(priority, 1))


def _confidence(classification: Classification) -> float:
    corroborating_signals = len(classification.evidence) - 1
    corroboration = min(MAX_CORROBORATION_BONUS, CORROBORATION_BONUS * corroborating_signals)
    return min(MAX_CONFIDENCE, BASE_CONFIDENCE + corroboration + _event_context(classification))


def _event_context(classification: Classification) -> float:
    primary = classification.primary
    anomaly_type = classification.rule.anomaly_type

    if primary.is_explained:
        duration_confirmed = primary.correlation.duration_matches is True
        return EVENT_CONTEXT_BONUS + (DURATION_MATCH_BONUS if duration_confirmed else 0)

    event = classification.related_event
    if event is not None and event.type in EVENTS_SUPPORTING_TYPE.get(anomaly_type, set()):
        return EVENT_CONTEXT_BONUS

    return 0.0


def _impact(classification: Classification) -> float:
    primary = classification.primary
    consumption_change = abs(primary.signal.magnitude) if primary.kind in CONSUMPTION_KINDS else 0.0
    return 0.5 + 0.5 * min(1.0, consumption_change / 100)


def _activity(classification: Classification) -> float:
    return 1.0 if classification.primary.signal.ended_at is None else RESOLVED_WEIGHT
