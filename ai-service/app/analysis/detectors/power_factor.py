from statistics import median

from app.analysis.detectors.base import intermittent_episode_end
from app.analysis.series import MeterSeries
from app.analysis.stats import percent_change
from app.domain.enums import SignalKind
from app.domain.models import Signal

MIN_POWER_FACTOR = 0.80
MIN_OCCURRENCES = 3


class PowerFactorDetector:
    def detect(self, series: MeterSeries) -> list[Signal]:
        monitored = [reading for reading in series.readings if reading.timestamp >= series.baseline_end]
        low = [reading for reading in monitored if reading.power_factor < MIN_POWER_FACTOR]
        if len(low) < MIN_OCCURRENCES:
            return []

        hours_since_first = [reading for reading in monitored if reading.timestamp >= low[0].timestamp]
        persistence = len(low) / len(hours_since_first) * 100

        baseline = series.baseline_power_factor
        observed = median(reading.power_factor for reading in low)
        lowest = min(reading.power_factor for reading in low)

        return [
            Signal(
                kind=SignalKind.LOW_POWER_FACTOR,
                started_at=low[0].timestamp,
                ended_at=intermittent_episode_end(series, low[-1].timestamp),
                magnitude=round(percent_change(baseline, observed), 1),
                baseline_value=round(baseline, 3),
                observed_value=round(observed, 3),
                description=(
                    f"Power factor below {MIN_POWER_FACTOR} in {len(low)} of {len(hours_since_first)} hours "
                    f"({persistence:.0f}% persistence, lowest {lowest:.2f}, baseline {baseline:.2f})"
                ),
            )
        ]
