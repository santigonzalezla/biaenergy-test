from datetime import datetime, timedelta
from itertools import pairwise
from statistics import median

from app.analysis.series import MeterSeries
from app.analysis.stats import percent_change
from app.domain.enums import SignalKind
from app.domain.models import ReadingInput, Signal

TOLERANCE_PCT = 5.0
MAX_STEP_PCT = 5.0
MIN_OCCURRENCES = 3
ACTIVE_WINDOW = timedelta(hours=24)


class VoltageDetector:
    def detect(self, series: MeterSeries) -> list[Signal]:
        monitored = [reading for reading in series.readings if reading.timestamp >= series.baseline_end]
        if len(monitored) < 2:
            return []

        signals = [self._out_of_range(series, monitored), self._instability(series, monitored)]
        return [signal for signal in signals if signal is not None]

    def _out_of_range(self, series: MeterSeries, readings: list[ReadingInput]) -> Signal | None:
        nominal = series.meter.nominal_voltage
        outside = [reading for reading in readings if abs(percent_change(nominal, reading.voltage)) > TOLERANCE_PCT]
        if len(outside) < MIN_OCCURRENCES:
            return None

        worst = max(outside, key=lambda reading: abs(reading.voltage - nominal))
        deviation = percent_change(nominal, worst.voltage)

        return Signal(
            kind=SignalKind.VOLTAGE_OUT_OF_RANGE,
            started_at=outside[0].timestamp,
            ended_at=self._episode_end(series, outside[-1].timestamp),
            magnitude=round(deviation, 1),
            baseline_value=nominal,
            observed_value=worst.voltage,
            description=(
                f"{len(outside)} readings outside the ±{TOLERANCE_PCT:.0f}% tolerance of {nominal:.0f} V "
                f"(worst {worst.voltage:.1f} V, {deviation:+.1f}%)"
            ),
        )

    def _instability(self, series: MeterSeries, readings: list[ReadingInput]) -> Signal | None:
        max_step = series.meter.nominal_voltage * MAX_STEP_PCT / 100
        steps = [(current, abs(current.voltage - previous.voltage)) for previous, current in pairwise(readings)]
        jumps = [(reading, step) for reading, step in steps if step > max_step]
        if len(jumps) < MIN_OCCURRENCES:
            return None

        largest = max(step for _, step in jumps)
        typical = median(step for _, step in steps)

        return Signal(
            kind=SignalKind.VOLTAGE_INSTABILITY,
            started_at=jumps[0][0].timestamp,
            ended_at=self._episode_end(series, jumps[-1][0].timestamp),
            magnitude=round(largest, 1),
            baseline_value=round(typical, 2),
            observed_value=round(largest, 2),
            description=(
                f"{len(jumps)} hour-to-hour voltage jumps above {max_step:.0f} V "
                f"(largest {largest:.1f} V, typical step {typical:.1f} V)"
            ),
        )

    @staticmethod
    def _episode_end(series: MeterSeries, last_occurrence: datetime) -> datetime | None:
        if series.last_timestamp - last_occurrence <= ACTIVE_WINDOW:
            return None
        return last_occurrence + timedelta(hours=1)
