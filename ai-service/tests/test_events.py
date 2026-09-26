from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

import pytest

from app.analysis.detectors.consumption_drop import ConsumptionDropDetector
from app.analysis.detectors.level_shift import LevelShiftDetector
from app.analysis.detectors.voltage import VoltageDetector
from app.analysis.events import correlate
from app.domain.enums import EventType, SignalKind
from app.domain.models import EventInput, Signal

BOGOTA = ZoneInfo("America/Bogota")
SIGNAL_START = datetime(2026, 9, 12, 14, tzinfo=BOGOTA)


def meter_events(dataset_request, code):
    return [event for event in dataset_request.events if event.meter_id == code]


@pytest.mark.parametrize(
    ("code", "detector", "expected_type", "expected_explains"),
    [
        pytest.param(
            "M-104", LevelShiftDetector(), EventType.OPERATIONAL_CHANGE, True, id="new line explains the surge"
        ),
        pytest.param(
            "M-106", ConsumptionDropDetector(), EventType.SCHEDULED_OUTAGE, True, id="outage explains the drop"
        ),
        pytest.param("M-109", LevelShiftDetector(), EventType.UNKNOWN, False, id="unknown event explains nothing"),
        pytest.param(
            "M-112", VoltageDetector(), EventType.DATA_QUALITY, False, id="data quality is related, not explaining"
        ),
    ],
)
def test_dataset_signals_meet_their_events(
    series_by_code, dataset_request, code, detector, expected_type, expected_explains
):
    signal = detector.detect(series_by_code[code])[0]

    correlation = correlate(signal, meter_events(dataset_request, code))

    assert correlation is not None
    assert correlation.event.type == expected_type
    assert correlation.explains is expected_explains


def test_announced_outage_duration_matches_the_observed_drop(series_by_code, dataset_request):
    [drop] = ConsumptionDropDetector().detect(series_by_code["M-106"])

    correlation = correlate(drop, meter_events(dataset_request, "M-106"))

    assert correlation.duration_matches is True


def signal(kind=SignalKind.CONSUMPTION_SURGE, ended_at=None):
    return Signal(
        kind=kind,
        started_at=SIGNAL_START,
        ended_at=ended_at,
        magnitude=50,
        baseline_value=40,
        observed_value=60,
        description="test signal",
    )


def event(event_type, hours_from_signal=0, description=""):
    return EventInput(
        id=f"{event_type}-{hours_from_signal}",
        meter_id="m",
        timestamp=SIGNAL_START + timedelta(hours=hours_from_signal),
        type=event_type,
        description=description,
    )


def test_events_outside_the_window_are_ignored():
    assert correlate(signal(), [event(EventType.OPERATIONAL_CHANGE, hours_from_signal=7)]) is None


def test_the_wrong_kind_of_event_does_not_explain():
    correlation = correlate(signal(), [event(EventType.SCHEDULED_OUTAGE)])

    assert correlation.explains is False


def test_an_explaining_event_wins_over_a_closer_one():
    correlation = correlate(
        signal(),
        [event(EventType.UNKNOWN, hours_from_signal=0), event(EventType.OPERATIONAL_CHANGE, hours_from_signal=3)],
    )

    assert correlation.event.type == EventType.OPERATIONAL_CHANGE
    assert correlation.explains is True


@pytest.mark.parametrize(
    ("observed_hours", "description", "expected"),
    [
        pytest.param(12, "Scheduled outage for 12 hours", True, id="same duration"),
        pytest.param(20, "Scheduled outage for 12 hours", False, id="drop lasted much longer than announced"),
        pytest.param(12, "Scheduled maintenance", None, id="no duration announced"),
    ],
)
def test_duration_consistency(observed_hours, description, expected):
    drop = signal(SignalKind.CONSUMPTION_DROP, ended_at=SIGNAL_START + timedelta(hours=observed_hours))

    correlation = correlate(drop, [event(EventType.SCHEDULED_OUTAGE, description=description)])

    assert correlation.duration_matches is expected
