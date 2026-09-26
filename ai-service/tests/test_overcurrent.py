from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

import pytest

from app.analysis.detectors.overcurrent import OvercurrentDetector
from app.analysis.series import build_series
from app.domain.enums import SignalKind
from app.domain.models import AnalysisRequest, MeterInput, ReadingInput

BOGOTA = ZoneInfo("America/Bogota")
OTHER_METERS = ["M-101", "M-102", "M-103", "M-104", "M-105", "M-106", "M-107", "M-108", "M-110", "M-111", "M-112"]


def test_detects_current_above_the_historical_peak_since_the_surge(series_by_code):
    [signal] = OvercurrentDetector().detect(series_by_code["M-109"])

    assert signal.kind == SignalKind.OVERCURRENT
    assert signal.started_at == datetime(2026, 9, 12, 14, tzinfo=BOGOTA)
    assert signal.ended_at is None
    assert signal.observed_value > 1.5 * signal.baseline_value
    assert "historical peak" in signal.description


@pytest.mark.parametrize("code", OTHER_METERS)
def test_ignores_meters_within_their_historical_peak(series_by_code, code):
    assert OvercurrentDetector().detect(series_by_code[code]) == []


def synthetic_series(currents_after_baseline: list[float], max_current: float | None = None):
    start = datetime(2026, 9, 1, tzinfo=BOGOTA)
    currents = [100.0] * len(currents_after_baseline) + currents_after_baseline
    readings = [
        ReadingInput(
            meter_id="m",
            timestamp=start + timedelta(hours=hour),
            consumption_kwh=50,
            voltage=220,
            current=current,
            power_factor=0.9,
        )
        for hour, current in enumerate(currents)
    ]
    meter = MeterInput(id="m", code="M-900", max_current=max_current)
    [series] = build_series(AnalysisRequest(meters=[meter], readings=readings))
    return series


@pytest.mark.parametrize(
    ("currents", "max_current", "expected_signals"),
    [
        pytest.param([140, 140, 140, 140], None, 0, id="below 1.5x the peak"),
        pytest.param([160, 160, 160, 100], None, 1, id="above 1.5x the peak"),
        pytest.param([160, 160, 100, 100], None, 0, id="two readings are not enough"),
        pytest.param([130, 130, 130, 100], 120, 1, id="rated maximum overrides the historical peak"),
    ],
)
def test_overcurrent_rules(currents, max_current, expected_signals):
    signals = OvercurrentDetector().detect(synthetic_series(currents, max_current))

    assert len(signals) == expected_signals
