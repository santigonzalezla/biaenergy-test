from datetime import timedelta
from statistics import median

from app.analysis.series import MeterSeries
from app.analysis.stats import percent_change
from app.domain.enums import SignalKind
from app.domain.models import ReadingInput, Signal

MIN_SURGE_PCT = 25.0
MIN_DURATION = timedelta(hours=24)
CUSUM_DRIFT = MIN_SURGE_PCT / 100 / 2
CUSUM_THRESHOLD = 1.0


class LevelShiftDetector:
    def detect(self, series: MeterSeries) -> list[Signal]:
        monitored = [reading for reading in series.readings if reading.timestamp >= series.baseline_end]
        change_index = self._find_upward_change(series, monitored)
        if change_index is None:
            return []

        shifted = monitored[change_index:]
        if shifted[-1].timestamp - shifted[0].timestamp < MIN_DURATION:
            return []

        expected = median(series.expected_kwh(reading) for reading in shifted)
        observed = median(reading.consumption_kwh for reading in shifted)
        change = percent_change(expected, observed)
        if change < MIN_SURGE_PCT:
            return []

        return [
            Signal(
                kind=SignalKind.CONSUMPTION_SURGE,
                started_at=shifted[0].timestamp,
                ended_at=None,
                magnitude=round(change, 1),
                baseline_value=round(expected, 2),
                observed_value=round(observed, 2),
                description=f"Sustained consumption increase of {change:.1f}% over the hourly baseline",
            )
        ]

    @staticmethod
    def _find_upward_change(series: MeterSeries, readings: list[ReadingInput]) -> int | None:
        cumulative = 0.0
        run_start: int | None = None

        for index, reading in enumerate(readings):
            expected = series.expected_kwh(reading)
            if expected <= 0:
                continue

            relative_deviation = (reading.consumption_kwh - expected) / expected
            cumulative = max(0.0, cumulative + relative_deviation - CUSUM_DRIFT)

            if cumulative == 0:
                run_start = None
            elif run_start is None:
                run_start = index

            if cumulative > CUSUM_THRESHOLD:
                return run_start

        return None
