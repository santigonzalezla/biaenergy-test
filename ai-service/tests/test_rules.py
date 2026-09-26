from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

import pytest

from app.analysis.detectors.consumption_drop import ConsumptionDropDetector
from app.analysis.detectors.level_shift import LevelShiftDetector
from app.analysis.detectors.overcurrent import OvercurrentDetector
from app.analysis.detectors.power_factor import PowerFactorDetector
from app.analysis.detectors.voltage import VoltageDetector
from app.analysis.events import EventCorrelation, correlate
from app.analysis.rules import Evidence, classify
from app.domain.enums import AnomalyType, EventType, Severity, SignalKind
from app.domain.models import EventInput, Signal

BOGOTA = ZoneInfo("America/Bogota")
START = datetime(2026, 9, 12, 14, tzinfo=BOGOTA)
DETECTORS = [
    LevelShiftDetector(),
    ConsumptionDropDetector(),
    VoltageDetector(),
    PowerFactorDetector(),
    OvercurrentDetector(),
]
NORMAL_METERS = ["M-101", "M-102", "M-103", "M-105", "M-107", "M-108", "M-110", "M-111"]


def dataset_evidence(series_by_code, dataset_request, code):
    events = [event for event in dataset_request.events if event.meter_id == code]
    signals = [signal for detector in DETECTORS for signal in detector.detect(series_by_code[code])]
    return [Evidence(signal=signal, correlation=correlate(signal, events)) for signal in signals]


@pytest.mark.parametrize(
    ("code", "rule_id", "anomaly_type", "severity", "signal_kinds"),
    [
        pytest.param(
            "M-109",
            "R4_UNEXPLAINED_SURGE",
            AnomalyType.REAL_ANOMALY,
            Severity.HIGH,
            [SignalKind.CONSUMPTION_SURGE, SignalKind.LOW_POWER_FACTOR, SignalKind.OVERCURRENT],
            id="M-109 real anomaly",
        ),
        pytest.param(
            "M-104",
            "R3_EXPLAINED_SURGE",
            AnomalyType.EXPLAINABLE_ANOMALY,
            Severity.MEDIUM,
            [SignalKind.CONSUMPTION_SURGE],
            id="M-104 explainable anomaly",
        ),
        pytest.param(
            "M-106",
            "R2_SCHEDULED_OUTAGE",
            AnomalyType.FALSE_POSITIVE,
            Severity.LOW,
            [SignalKind.CONSUMPTION_DROP],
            id="M-106 false positive",
        ),
        pytest.param(
            "M-112",
            "R1_MEASUREMENT_FAULT",
            AnomalyType.DATA_QUALITY,
            Severity.HIGH,
            [SignalKind.VOLTAGE_OUT_OF_RANGE, SignalKind.VOLTAGE_INSTABILITY, SignalKind.LOW_POWER_FACTOR],
            id="M-112 data quality",
        ),
    ],
)
def test_classifies_the_four_dataset_cases(
    series_by_code, dataset_request, code, rule_id, anomaly_type, severity, signal_kinds
):
    [classification] = classify(dataset_evidence(series_by_code, dataset_request, code))

    assert classification.rule.rule_id == rule_id
    assert classification.rule.anomaly_type == anomaly_type
    assert classification.severity == severity
    assert [item.kind for item in classification.evidence] == signal_kinds


@pytest.mark.parametrize("code", NORMAL_METERS)
def test_normal_meters_produce_no_findings(series_by_code, dataset_request, code):
    assert classify(dataset_evidence(series_by_code, dataset_request, code)) == []


def test_related_event_is_reported(series_by_code, dataset_request):
    [classification] = classify(dataset_evidence(series_by_code, dataset_request, "M-109"))

    assert classification.related_event.type == EventType.UNKNOWN


# Synthetic evidence


def evidence(kind, magnitude=60.0, ended=False, event_type=None, explains=False, duration_matches=None):
    signal = Signal(
        kind=kind,
        started_at=START,
        ended_at=START + timedelta(hours=12) if ended else None,
        magnitude=magnitude,
        baseline_value=10,
        observed_value=20,
        description=kind,
    )
    correlation = None
    if event_type is not None:
        event = EventInput(id="e", meter_id="m", timestamp=START, type=event_type, description="")
        correlation = EventCorrelation(event=event, explains=explains, duration_matches=duration_matches)
    return Evidence(signal=signal, correlation=correlation)


@pytest.mark.parametrize(
    ("items", "expected_rules"),
    [
        pytest.param(
            [evidence(SignalKind.CONSUMPTION_DROP, magnitude=-80, ended=True)],
            ["R5_UNEXPLAINED_DROP"],
            id="drop without an event is real",
        ),
        pytest.param(
            [
                evidence(
                    SignalKind.CONSUMPTION_DROP,
                    magnitude=-80,
                    ended=True,
                    event_type=EventType.SCHEDULED_OUTAGE,
                    explains=True,
                    duration_matches=False,
                )
            ],
            ["R5_UNEXPLAINED_DROP"],
            id="drop longer than announced is real",
        ),
        pytest.param(
            [
                evidence(SignalKind.CONSUMPTION_SURGE, event_type=EventType.OPERATIONAL_CHANGE, explains=True),
                evidence(SignalKind.LOW_POWER_FACTOR),
            ],
            ["R4_UNEXPLAINED_SURGE"],
            id="explained surge with degraded power factor is real",
        ),
        pytest.param(
            [evidence(SignalKind.CONSUMPTION_SURGE, magnitude=30)],
            ["R4_UNEXPLAINED_SURGE"],
            id="small unexplained surge is still real",
        ),
        pytest.param(
            [evidence(SignalKind.LOW_POWER_FACTOR)],
            ["R6_EQUIPMENT_FAULT"],
            id="power factor alone is an equipment fault",
        ),
        pytest.param(
            [
                evidence(SignalKind.CONSUMPTION_DROP, magnitude=-80, ended=True),
                evidence(SignalKind.CONSUMPTION_SURGE),
            ],
            ["R4_UNEXPLAINED_SURGE", "R5_UNEXPLAINED_DROP"],
            id="independent problems produce independent findings",
        ),
    ],
)
def test_rule_selection(items, expected_rules):
    assert [classification.rule.rule_id for classification in classify(items)] == expected_rules


def test_small_unexplained_surge_is_medium_severity():
    [classification] = classify([evidence(SignalKind.CONSUMPTION_SURGE, magnitude=30)])

    assert classification.severity == Severity.MEDIUM
