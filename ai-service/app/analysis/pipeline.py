from collections import defaultdict
from collections.abc import Sequence
from datetime import datetime
from typing import Protocol

from app.analysis.detectors.base import Detector
from app.analysis.detectors.consumption_drop import ConsumptionDropDetector
from app.analysis.detectors.level_shift import LevelShiftDetector
from app.analysis.detectors.overcurrent import OvercurrentDetector
from app.analysis.detectors.power_factor import PowerFactorDetector
from app.analysis.detectors.voltage import VoltageDetector
from app.analysis.events import correlate
from app.analysis.rules import Classification, Evidence, classify
from app.analysis.scoring import score
from app.analysis.series import MeterSeries, build_series
from app.analysis.stats import percent_change
from app.domain.enums import SignalKind
from app.domain.models import (
    AnalysisRequest,
    AnalysisResult,
    ChangedVariable,
    EventInput,
    EventReference,
    Explanation,
    Finding,
    Signal,
)

VARIABLE_BY_SIGNAL: dict[SignalKind, str] = {
    SignalKind.CONSUMPTION_SURGE: "consumption_kwh",
    SignalKind.CONSUMPTION_DROP: "consumption_kwh",
    SignalKind.VOLTAGE_OUT_OF_RANGE: "voltage",
    SignalKind.VOLTAGE_INSTABILITY: "voltage_step",
    SignalKind.LOW_POWER_FACTOR: "power_factor",
    SignalKind.OVERCURRENT: "current",
}


class Explainer(Protocol):
    def explain(self, finding: Finding) -> Explanation: ...


def default_detectors() -> list[Detector]:
    return [
        LevelShiftDetector(),
        ConsumptionDropDetector(),
        VoltageDetector(),
        PowerFactorDetector(),
        OvercurrentDetector(),
    ]


class AnomalyAnalyzer:
    def __init__(self, explainer: Explainer, detectors: Sequence[Detector] | None = None):
        self._explainer = explainer
        self._detectors = list(detectors) if detectors is not None else default_detectors()

    def analyze(self, request: AnalysisRequest) -> AnalysisResult:
        events_by_meter: dict[str, list[EventInput]] = defaultdict(list)
        for event in request.events:
            events_by_meter[event.meter_id].append(event)

        findings = [
            finding
            for series in build_series(request)
            for finding in self._analyze_meter(series, events_by_meter[series.meter.id])
        ]
        findings.sort(key=lambda finding: finding.priority_score, reverse=True)

        return AnalysisResult(meters_analyzed=len(request.meters), findings=findings)

    def _analyze_meter(self, series: MeterSeries, events: list[EventInput]) -> list[Finding]:
        signals = [signal for detector in self._detectors for signal in detector.detect(series)]
        evidence = [Evidence(signal=signal, correlation=correlate(signal, events)) for signal in signals]
        return [self._to_finding(series, classification) for classification in classify(evidence)]

    def _to_finding(self, series: MeterSeries, classification: Classification) -> Finding:
        scored = score(classification)
        primary = classification.primary.signal
        signals = [item.signal for item in classification.evidence]
        window_start = min(signal.started_at for signal in signals)
        expected_kwh, observed_kwh = _consumption_in_window(series, window_start, primary.ended_at)
        related_event = classification.related_event

        finding = Finding(
            meter_id=series.meter.id,
            meter_code=series.meter.code,
            type=classification.rule.anomaly_type,
            severity=classification.severity,
            rule_id=classification.rule.rule_id,
            confidence=scored.confidence,
            priority_score=scored.priority,
            detected_at=primary.started_at,
            window_start=window_start,
            window_end=primary.ended_at,
            baseline_kwh=round(expected_kwh, 2),
            current_kwh=round(observed_kwh, 2),
            variation_pct=round(percent_change(expected_kwh, observed_kwh), 1),
            changed_variables=[_changed_variable(signal) for signal in signals],
            signals=signals,
            related_event=_event_reference(related_event) if related_event else None,
        )

        explanation = self._explainer.explain(finding)
        return finding.model_copy(update=explanation.model_dump())


def _consumption_in_window(series: MeterSeries, start: datetime, end: datetime | None) -> tuple[float, float]:
    window = [
        reading
        for reading in series.readings
        if reading.timestamp >= start and (end is None or reading.timestamp < end)
    ]
    expected = sum(series.expected_kwh(reading) for reading in window) / len(window)
    observed = sum(reading.consumption_kwh for reading in window) / len(window)
    return expected, observed


def _event_reference(event: EventInput) -> EventReference:
    return EventReference(id=event.id, type=event.type, timestamp=event.timestamp, description=event.description)


def _changed_variable(signal: Signal) -> ChangedVariable:
    return ChangedVariable(
        name=VARIABLE_BY_SIGNAL[signal.kind],
        baseline=signal.baseline_value,
        observed=signal.observed_value,
        change_pct=round(percent_change(signal.baseline_value, signal.observed_value), 1),
    )
