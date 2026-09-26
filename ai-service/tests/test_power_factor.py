from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

import pytest

from app.analysis.detectors.power_factor import PowerFactorDetector
from app.analysis.series import build_series
from app.domain.enums import SignalKind
from app.domain.models import AnalysisRequest, MeterInput, ReadingInput

BOGOTA = ZoneInfo("America/Bogota")
HEALTHY_METERS = ["M-101", "M-102", "M-103", "M-104", "M-105", "M-106", "M-107", "M-108", "M-110", "M-111"]


def test_sustained_low_power_factor_since_the_surge(series_by_code):
    [signal] = PowerFactorDetector().detect(series_by_code["M-109"])

    assert signal.kind == SignalKind.LOW_POWER_FACTOR
    assert signal.started_at == datetime(2026, 9, 12, 14, tzinfo=BOGOTA)
    assert signal.ended_at is None
    assert signal.observed_value < 0.8 < signal.baseline_value
    assert "100% persistence" in signal.description


def test_intermittent_low_power_factor_of_the_faulty_meter(series_by_code):
    [signal] = PowerFactorDetector().detect(series_by_code["M-112"])

    assert signal.started_at.date().isoformat() == "2026-09-13"
    assert signal.ended_at is None
    assert "100% persistence" not in signal.description


@pytest.mark.parametrize("code", HEALTHY_METERS)
def test_ignores_meters_with_healthy_power_factor(series_by_code, code):
    assert PowerFactorDetector().detect(series_by_code[code]) == []


def test_two_low_readings_are_not_enough():
    start = datetime(2026, 9, 1, tzinfo=BOGOTA)
    power_factors = [0.95] * 12 + [0.95, 0.70, 0.70, 0.95, 0.95, 0.95]
    readings = [
        ReadingInput(
            meter_id="m",
            timestamp=start + timedelta(hours=hour),
            consumption_kwh=50,
            voltage=220,
            current=100,
            power_factor=power_factor,
        )
        for hour, power_factor in enumerate(power_factors)
    ]
    [series] = build_series(AnalysisRequest(meters=[MeterInput(id="m", code="M-900")], readings=readings))

    assert PowerFactorDetector().detect(series) == []
