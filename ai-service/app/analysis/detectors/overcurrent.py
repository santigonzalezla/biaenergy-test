from statistics import median

from app.analysis.detectors.base import intermittent_episode_end
from app.analysis.series import MeterSeries
from app.analysis.stats import percent_change
from app.domain.enums import SignalKind
from app.domain.models import Signal

PEAK_MARGIN = 1.5
MIN_OCCURRENCES = 3


class OvercurrentDetector:
    def detect(self, series: MeterSeries) -> list[Signal]:
        limit, reference = self._current_limit(series)
        monitored = [reading for reading in series.readings if reading.timestamp >= series.baseline_end]
        over_limit = [reading for reading in monitored if reading.current > limit]
        if len(over_limit) < MIN_OCCURRENCES:
            return []

        observed = median(reading.current for reading in over_limit)
        highest = max(reading.current for reading in over_limit)

        return [
            Signal(
                kind=SignalKind.OVERCURRENT,
                started_at=over_limit[0].timestamp,
                ended_at=intermittent_episode_end(series, over_limit[-1].timestamp),
                magnitude=round(percent_change(reference, observed), 1),
                baseline_value=round(reference, 2),
                observed_value=round(observed, 2),
                description=(
                    f"{len(over_limit)} readings above the {limit:.1f} A limit "
                    f"({self._limit_origin(series)}; highest {highest:.1f} A)"
                ),
            )
        ]

    @staticmethod
    def _current_limit(series: MeterSeries) -> tuple[float, float]:
        if series.meter.max_current is not None:
            return series.meter.max_current, series.meter.max_current

        historical_peak = max(reading.current for reading in series.baseline_readings)
        return historical_peak * PEAK_MARGIN, historical_peak

    @staticmethod
    def _limit_origin(series: MeterSeries) -> str:
        if series.meter.max_current is not None:
            return "rated maximum current"
        return f"{PEAK_MARGIN}x the historical peak"
