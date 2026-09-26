from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

import pytest

from app.analysis.detectors.consumption_drop import ConsumptionDropDetector
from app.analysis.series import build_series
from app.domain.enums import SignalKind
from app.domain.models import AnalysisRequest, MeterInput, ReadingInput

BOGOTA = ZoneInfo("America/Bogota")
NORMAL_METERS = ["M-101", "M-102", "M-103", "M-105", "M-107", "M-108", "M-110", "M-111"]


def test_detects_the_scheduled_outage_with_its_start_and_recovery(series_by_code):
    [signal] = ConsumptionDropDetector().detect(series_by_code["M-106"])

    assert signal.kind == SignalKind.CONSUMPTION_DROP
    assert signal.started_at == datetime(2026, 9, 8, 0, tzinfo=BOGOTA)
    assert signal.ended_at == datetime(2026, 9, 8, 12, tzinfo=BOGOTA)
    assert signal.magnitude < -75
    assert "12 consecutive hours" in signal.description


@pytest.mark.parametrize("code", [*NORMAL_METERS, "M-104", "M-109", "M-112"])
def test_ignores_meters_without_a_drop(series_by_code, code):
    assert ConsumptionDropDetector().detect(series_by_code[code]) == []


def synthetic_series(low_hours: set[int], total_hours: int = 48):
    start = datetime(2026, 9, 1, tzinfo=BOGOTA)
    readings = [
        ReadingInput(
            meter_id="m",
            timestamp=start + timedelta(hours=hour),
            consumption_kwh=5 if hour in low_hours else 50,
            voltage=220,
            current=100,
            power_factor=0.9,
        )
        for hour in range(total_hours)
    ]
    [series] = build_series(AnalysisRequest(meters=[MeterInput(id="m", code="M-900")], readings=readings))
    return series


@pytest.mark.parametrize(
    ("low_hours", "expected_signals"),
    [
        pytest.param({30, 31}, 0, id="two low hours are a blip"),
        pytest.param({30, 31, 32}, 1, id="three low hours are a drop"),
        pytest.param({26, 27, 28, 40, 41, 42}, 2, id="two separate drops"),
    ],
)
def test_requires_consecutive_low_hours(low_hours, expected_signals):
    assert len(ConsumptionDropDetector().detect(synthetic_series(low_hours))) == expected_signals


def test_drop_still_active_at_the_end_has_no_end():
    [signal] = ConsumptionDropDetector().detect(synthetic_series({44, 45, 46, 47}))

    assert signal.ended_at is None
