from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

import pytest

from app.analysis.detectors.voltage import VoltageDetector
from app.analysis.series import build_series
from app.domain.enums import SignalKind
from app.domain.models import AnalysisRequest, MeterInput, ReadingInput

BOGOTA = ZoneInfo("America/Bogota")
OTHER_METERS = ["M-101", "M-102", "M-103", "M-104", "M-105", "M-106", "M-107", "M-108", "M-109", "M-110", "M-111"]


def signals_by_kind(series):
    return {signal.kind: signal for signal in VoltageDetector().detect(series)}


def test_detects_both_voltage_problems_of_the_faulty_meter(series_by_code):
    signals = signals_by_kind(series_by_code["M-112"])

    out_of_range = signals[SignalKind.VOLTAGE_OUT_OF_RANGE]
    assert out_of_range.started_at.date().isoformat() == "2026-09-13"
    assert out_of_range.ended_at is None
    assert abs(out_of_range.magnitude) > 5
    assert out_of_range.baseline_value == 220

    instability = signals[SignalKind.VOLTAGE_INSTABILITY]
    assert instability.started_at.date().isoformat() == "2026-09-13"
    assert instability.observed_value > 20
    assert instability.baseline_value < 5


@pytest.mark.parametrize("code", OTHER_METERS)
def test_ignores_meters_with_healthy_voltage(series_by_code, code):
    assert VoltageDetector().detect(series_by_code[code]) == []


def synthetic_series(voltages_after_baseline: list[float], nominal: float = 220):
    start = datetime(2026, 9, 1, tzinfo=BOGOTA)
    voltages = [nominal] * len(voltages_after_baseline) + voltages_after_baseline
    readings = [
        ReadingInput(
            meter_id="m",
            timestamp=start + timedelta(hours=hour),
            consumption_kwh=50,
            voltage=voltage,
            current=100,
            power_factor=0.9,
        )
        for hour, voltage in enumerate(voltages)
    ]
    meter = MeterInput(id="m", code="M-900", nominal_voltage=nominal)
    [series] = build_series(AnalysisRequest(meters=[meter], readings=readings))
    return series


@pytest.mark.parametrize(
    ("voltages", "expected_kinds"),
    [
        pytest.param([220, 232, 232, 220, 220, 220], set(), id="two readings and two jumps are not enough"),
        pytest.param([233, 233, 233, 220, 220, 220], {SignalKind.VOLTAGE_OUT_OF_RANGE}, id="steady overvoltage"),
        pytest.param(
            [220, 208, 220, 208, 220, 220],
            {SignalKind.VOLTAGE_INSTABILITY},
            id="jumps inside the tolerance band",
        ),
        pytest.param(
            [440, 440, 440, 440, 440, 440],
            set(),
            id="the tolerance follows the meter nominal voltage",
        ),
    ],
)
def test_voltage_rules(voltages, expected_kinds):
    nominal = 440 if voltages[0] == 440 else 220

    assert set(signals_by_kind(synthetic_series(voltages, nominal))) == expected_kinds
