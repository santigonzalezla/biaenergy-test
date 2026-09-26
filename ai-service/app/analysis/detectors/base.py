from typing import Protocol

from app.analysis.series import MeterSeries
from app.domain.models import Signal


class Detector(Protocol):
    def detect(self, series: MeterSeries) -> list[Signal]: ...
