from collections.abc import Callable, Sequence
from dataclasses import dataclass

from app.analysis.events import EventCorrelation
from app.domain.enums import AnomalyType, Severity, SignalKind
from app.domain.models import EventInput, Signal

CONSUMPTION_KINDS = {SignalKind.CONSUMPTION_SURGE, SignalKind.CONSUMPTION_DROP}
VOLTAGE_KINDS = {SignalKind.VOLTAGE_OUT_OF_RANGE, SignalKind.VOLTAGE_INSTABILITY}
EQUIPMENT_KINDS = {SignalKind.LOW_POWER_FACTOR, SignalKind.OVERCURRENT}
SEVERE_SURGE_PCT = 50.0


@dataclass(frozen=True)
class Evidence:
    signal: Signal
    correlation: EventCorrelation | None

    @property
    def kind(self) -> SignalKind:
        return self.signal.kind

    @property
    def is_explained(self) -> bool:
        return self.correlation is not None and self.correlation.explains


@dataclass(frozen=True)
class Match:
    evidence: tuple[Evidence, ...]
    severity: Severity


@dataclass(frozen=True)
class Rule:
    rule_id: str
    anomaly_type: AnomalyType
    summary: str
    matcher: Callable[[Sequence[Evidence]], Match | None]


@dataclass(frozen=True)
class Classification:
    rule: Rule
    severity: Severity
    evidence: tuple[Evidence, ...]

    @property
    def primary(self) -> Evidence:
        return self.evidence[0]

    @property
    def related_event(self) -> EventInput | None:
        correlated = [item.correlation for item in self.evidence if item.correlation is not None]
        return correlated[0].event if correlated else None


def classify(evidence: Sequence[Evidence]) -> list[Classification]:
    remaining = list(evidence)
    classifications: list[Classification] = []

    for rule in RULES:
        while (match := rule.matcher(remaining)) is not None:
            classifications.append(Classification(rule=rule, severity=match.severity, evidence=match.evidence))
            remaining = [item for item in remaining if item not in match.evidence]

    return classifications


def of_kinds(evidence: Sequence[Evidence], kinds: set[SignalKind]) -> list[Evidence]:
    return [item for item in evidence if item.kind in kinds]


# Matchers


def measurement_fault(evidence: Sequence[Evidence]) -> Match | None:
    voltage = of_kinds(evidence, VOLTAGE_KINDS)
    if not voltage or of_kinds(evidence, CONSUMPTION_KINDS):
        return None
    return Match(evidence=(*voltage, *of_kinds(evidence, EQUIPMENT_KINDS)), severity=Severity.HIGH)


def scheduled_outage(evidence: Sequence[Evidence]) -> Match | None:
    for drop in of_kinds(evidence, {SignalKind.CONSUMPTION_DROP}):
        recovered = drop.signal.ended_at is not None
        duration_consistent = drop.correlation is not None and drop.correlation.duration_matches is not False
        if drop.is_explained and recovered and duration_consistent:
            return Match(evidence=(drop,), severity=Severity.LOW)
    return None


def explained_surge(evidence: Sequence[Evidence]) -> Match | None:
    if of_kinds(evidence, {SignalKind.LOW_POWER_FACTOR}):
        return None
    for surge in of_kinds(evidence, {SignalKind.CONSUMPTION_SURGE}):
        if surge.is_explained:
            return Match(evidence=(surge, *of_kinds(evidence, {SignalKind.OVERCURRENT})), severity=Severity.MEDIUM)
    return None


def unexplained_surge(evidence: Sequence[Evidence]) -> Match | None:
    surges = of_kinds(evidence, {SignalKind.CONSUMPTION_SURGE})
    if not surges:
        return None

    surge = surges[0]
    equipment = of_kinds(evidence, EQUIPMENT_KINDS)
    severe = bool(equipment) or surge.signal.magnitude >= SEVERE_SURGE_PCT
    return Match(evidence=(surge, *equipment), severity=Severity.HIGH if severe else Severity.MEDIUM)


def unexplained_drop(evidence: Sequence[Evidence]) -> Match | None:
    drops = of_kinds(evidence, {SignalKind.CONSUMPTION_DROP})
    if not drops:
        return None

    drop = drops[0]
    ongoing = drop.signal.ended_at is None
    return Match(evidence=(drop,), severity=Severity.HIGH if ongoing else Severity.MEDIUM)


def equipment_fault(evidence: Sequence[Evidence]) -> Match | None:
    equipment = of_kinds(evidence, EQUIPMENT_KINDS)
    if not equipment:
        return None
    return Match(evidence=tuple(equipment), severity=Severity.MEDIUM)


RULES: tuple[Rule, ...] = (
    Rule(
        rule_id="R1_MEASUREMENT_FAULT",
        anomaly_type=AnomalyType.DATA_QUALITY,
        summary="Abnormal voltage while consumption stays stable: the measurement is unreliable",
        matcher=measurement_fault,
    ),
    Rule(
        rule_id="R2_SCHEDULED_OUTAGE",
        anomaly_type=AnomalyType.FALSE_POSITIVE,
        summary="Consumption drop that matches a planned outage or maintenance and already recovered",
        matcher=scheduled_outage,
    ),
    Rule(
        rule_id="R3_EXPLAINED_SURGE",
        anomaly_type=AnomalyType.EXPLAINABLE_ANOMALY,
        summary="Sustained increase justified by an operational change, with healthy electrical behaviour",
        matcher=explained_surge,
    ),
    Rule(
        rule_id="R4_UNEXPLAINED_SURGE",
        anomaly_type=AnomalyType.REAL_ANOMALY,
        summary="Sustained increase without a known cause, or with degraded electrical behaviour",
        matcher=unexplained_surge,
    ),
    Rule(
        rule_id="R5_UNEXPLAINED_DROP",
        anomaly_type=AnomalyType.REAL_ANOMALY,
        summary="Consumption drop without a planned event, or longer than announced",
        matcher=unexplained_drop,
    ),
    Rule(
        rule_id="R6_EQUIPMENT_FAULT",
        anomaly_type=AnomalyType.REAL_ANOMALY,
        summary="Low power factor or overcurrent without a consumption change",
        matcher=equipment_fault,
    ),
)
