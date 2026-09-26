from datetime import datetime
from zoneinfo import ZoneInfo

import pytest

from app.analysis.detectors.level_shift import LevelShiftDetector
from app.domain.enums import SignalKind

BOGOTA = ZoneInfo("America/Bogota")
NORMAL_METERS = ["M-101", "M-102", "M-103", "M-105", "M-107", "M-108", "M-110", "M-111"]


@pytest.mark.parametrize(
    ("code", "expected_start", "min_change_pct"),
    [
        pytest.param("M-109", datetime(2026, 9, 12, 14, tzinfo=BOGOTA), 100, id="M-109 unexplained surge"),
        pytest.param("M-104", datetime(2026, 9, 11, 0, tzinfo=BOGOTA), 40, id="M-104 new production line"),
    ],
)
def test_detects_sustained_surges_at_the_exact_change_point(series_by_code, code, expected_start, min_change_pct):
    [signal] = LevelShiftDetector().detect(series_by_code[code])

    assert signal.kind == SignalKind.CONSUMPTION_SURGE
    assert signal.started_at == expected_start
    assert signal.ended_at is None
    assert signal.magnitude >= min_change_pct
    assert signal.observed_value > signal.baseline_value


@pytest.mark.parametrize("code", [*NORMAL_METERS, "M-106", "M-112"])
def test_ignores_meters_without_a_sustained_surge(series_by_code, code):
    assert LevelShiftDetector().detect(series_by_code[code]) == []
