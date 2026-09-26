import csv
from datetime import datetime
from pathlib import Path
from zoneinfo import ZoneInfo

import pytest

from app.analysis.series import MeterSeries, build_series
from app.domain.models import AnalysisRequest, EventInput, MeterInput, ReadingInput

DATA_DIR = Path(__file__).resolve().parents[2] / "data"
SITE_TIMEZONE = "America/Bogota"


def parse_local(value: str) -> datetime:
    layout = "%Y-%m-%d %H:%M:%S" if value.count(":") == 2 else "%Y-%m-%d %H:%M"
    return datetime.strptime(value, layout).replace(tzinfo=ZoneInfo(SITE_TIMEZONE))


def load_rows(filename: str) -> list[dict[str, str]]:
    with (DATA_DIR / filename).open(newline="") as file:
        return list(csv.DictReader(file))


@pytest.fixture(scope="session")
def dataset_request() -> AnalysisRequest:
    reading_rows = load_rows("readings.csv")
    event_rows = load_rows("events.csv")
    meter_codes = sorted({row["meter_id"] for row in reading_rows})

    return AnalysisRequest(
        timezone=SITE_TIMEZONE,
        meters=[MeterInput(id=code, code=code) for code in meter_codes],
        readings=[
            ReadingInput(
                meter_id=row["meter_id"],
                timestamp=parse_local(row["timestamp"]),
                consumption_kwh=float(row["consumption_kwh"]),
                voltage=float(row["voltage_v"]),
                current=float(row["current_a"]),
                power_factor=float(row["power_factor"]),
            )
            for row in reading_rows
        ],
        events=[
            EventInput(
                id=f"event-{index}",
                meter_id=row["meter_id"],
                timestamp=parse_local(row["event_timestamp"]),
                type=row["event_type"],
                description=row["description"],
            )
            for index, row in enumerate(event_rows)
        ],
    )


@pytest.fixture(scope="session")
def series_by_code(dataset_request: AnalysisRequest) -> dict[str, MeterSeries]:
    return {series.meter.code: series for series in build_series(dataset_request)}
