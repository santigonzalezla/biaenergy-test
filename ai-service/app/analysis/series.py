from collections import defaultdict
from collections.abc import Callable
from dataclasses import dataclass
from datetime import datetime, timedelta
from functools import cached_property
from statistics import median
from zoneinfo import ZoneInfo

from app.analysis.stats import median_absolute_deviation
from app.domain.models import AnalysisRequest, MeterInput, ReadingInput

BASELINE_WINDOW = timedelta(days=7)


@dataclass(frozen=True)
class HourlyBaseline:
    median_kwh: float
    mad_kwh: float


@dataclass(frozen=True)
class MeterSeries:
    meter: MeterInput
    readings: tuple[ReadingInput, ...]
    timezone: ZoneInfo

    @property
    def first_timestamp(self) -> datetime:
        return self.readings[0].timestamp

    @property
    def last_timestamp(self) -> datetime:
        return self.readings[-1].timestamp

    @cached_property
    def baseline_end(self) -> datetime:
        half_span = (self.last_timestamp - self.first_timestamp) / 2
        return self.first_timestamp + min(BASELINE_WINDOW, half_span)

    @cached_property
    def baseline_readings(self) -> tuple[ReadingInput, ...]:
        return tuple(reading for reading in self.readings if reading.timestamp < self.baseline_end)

    @cached_property
    def hourly_baseline(self) -> dict[int, HourlyBaseline]:
        consumption_by_hour: dict[int, list[float]] = defaultdict(list)
        for reading in self.baseline_readings:
            consumption_by_hour[self.local_hour(reading)].append(reading.consumption_kwh)

        return {
            hour: HourlyBaseline(median_kwh=median(values), mad_kwh=median_absolute_deviation(values))
            for hour, values in consumption_by_hour.items()
        }

    @cached_property
    def baseline_kwh(self) -> float:
        return self._baseline_median(lambda reading: reading.consumption_kwh)

    @cached_property
    def baseline_voltage(self) -> float:
        return self._baseline_median(lambda reading: reading.voltage)

    @cached_property
    def baseline_current(self) -> float:
        return self._baseline_median(lambda reading: reading.current)

    @cached_property
    def baseline_power_factor(self) -> float:
        return self._baseline_median(lambda reading: reading.power_factor)

    def local_hour(self, reading: ReadingInput) -> int:
        return reading.timestamp.astimezone(self.timezone).hour

    def expected_kwh(self, reading: ReadingInput) -> float:
        hourly = self.hourly_baseline.get(self.local_hour(reading))
        return hourly.median_kwh if hourly else self.baseline_kwh

    def _baseline_median(self, value_of: Callable[[ReadingInput], float]) -> float:
        return median(value_of(reading) for reading in self.baseline_readings)


def build_series(request: AnalysisRequest) -> list[MeterSeries]:
    timezone = ZoneInfo(request.timezone)

    readings_by_meter: dict[str, list[ReadingInput]] = defaultdict(list)
    for reading in request.readings:
        readings_by_meter[reading.meter_id].append(reading)

    return [
        MeterSeries(
            meter=meter,
            readings=tuple(sorted(readings_by_meter[meter.id], key=lambda reading: reading.timestamp)),
            timezone=timezone,
        )
        for meter in request.meters
        if readings_by_meter[meter.id]
    ]
