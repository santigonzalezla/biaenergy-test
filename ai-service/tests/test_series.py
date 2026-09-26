from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

import pytest

from app.analysis.series import build_series
from app.domain.models import AnalysisRequest, MeterInput, ReadingInput

BOGOTA = ZoneInfo("America/Bogota")


def test_builds_one_sorted_series_per_meter(series_by_code):
    assert len(series_by_code) == 12

    for series in series_by_code.values():
        timestamps = [reading.timestamp for reading in series.readings]
        assert len(timestamps) == 336
        assert timestamps == sorted(timestamps)


def test_baseline_is_the_first_week(series_by_code):
    series = series_by_code["M-109"]

    assert len(series.baseline_readings) == 168
    assert all(reading.timestamp < datetime(2026, 9, 8, tzinfo=BOGOTA) for reading in series.baseline_readings)


@pytest.mark.parametrize(
    ("local_hour", "expected_median"),
    [
        pytest.param(3, 33.2, id="night base load"),
        pytest.param(8, 51.29, id="day shift"),
        pytest.param(14, 51.66, id="afternoon, when M-109 later jumps"),
        pytest.param(19, 44.21, id="evening shift"),
    ],
)
def test_hourly_baseline_follows_the_daily_profile(series_by_code, local_hour, expected_median):
    hourly = series_by_code["M-109"].hourly_baseline[local_hour]

    assert hourly.median_kwh == pytest.approx(expected_median)
    assert hourly.mad_kwh < 2


def test_expected_kwh_uses_the_local_hour(series_by_code):
    series = series_by_code["M-109"]
    surge_reading = next(
        reading for reading in series.readings if reading.timestamp == datetime(2026, 9, 12, 14, tzinfo=BOGOTA)
    )

    assert series.local_hour(surge_reading) == 14
    assert series.expected_kwh(surge_reading) == pytest.approx(51.66)
    assert surge_reading.consumption_kwh == pytest.approx(110.35)


def test_baseline_electrical_values(series_by_code):
    series = series_by_code["M-109"]

    assert series.baseline_voltage == pytest.approx(219.7, abs=0.1)
    assert series.baseline_current == pytest.approx(195.9, abs=0.1)
    assert series.baseline_power_factor == pytest.approx(0.94)


def test_short_series_uses_half_of_its_span_as_baseline():
    start = datetime(2026, 9, 1, tzinfo=BOGOTA)
    readings = [
        ReadingInput(
            meter_id="m",
            timestamp=start + timedelta(hours=hour),
            consumption_kwh=10,
            voltage=220,
            current=50,
            power_factor=0.9,
        )
        for hour in range(48)
    ]
    request = AnalysisRequest(meters=[MeterInput(id="m", code="M-900")], readings=readings)

    [series] = build_series(request)

    assert series.baseline_end == start + timedelta(hours=23, minutes=30)
    assert len(series.baseline_readings) == 24


def test_meters_without_readings_are_skipped():
    request = AnalysisRequest(meters=[MeterInput(id="empty", code="M-999")], readings=[])

    assert build_series(request) == []
