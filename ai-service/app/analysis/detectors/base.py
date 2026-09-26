from datetime import datetime, timedelta
from typing import Protocol

from app.analysis.series import MeterSeries
from app.domain.models import Signal

ACTIVE_WINDOW = timedelta(hours=24)
READING_INTERVAL = timedelta(hours=1)


class Detector(Protocol):
    def detect(self, series: MeterSeries) -> list[Signal]: ...


def intermittent_episode_end(series: MeterSeries, last_occurrence: datetime) -> datetime | None:
    if series.last_timestamp - last_occurrence <= ACTIVE_WINDOW:
        return None
    return last_occurrence + READING_INTERVAL
