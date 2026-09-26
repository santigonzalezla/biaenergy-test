from statistics import median

from app.analysis.series import MeterSeries
from app.analysis.stats import percent_change
from app.domain.enums import SignalKind
from app.domain.models import ReadingInput, Signal

MAX_RATIO_TO_EXPECTED = 0.5
MIN_CONSECUTIVE_HOURS = 3


class ConsumptionDropDetector:
    def detect(self, series: MeterSeries) -> list[Signal]:
        monitored = [reading for reading in series.readings if reading.timestamp >= series.baseline_end]
        signals: list[Signal] = []
        low_run: list[ReadingInput] = []

        for reading in monitored:
            if self._is_low(series, reading):
                low_run.append(reading)
                continue

            if len(low_run) >= MIN_CONSECUTIVE_HOURS:
                signals.append(self._to_signal(series, low_run, recovered_at=reading))
            low_run = []

        if len(low_run) >= MIN_CONSECUTIVE_HOURS:
            signals.append(self._to_signal(series, low_run, recovered_at=None))

        return signals

    @staticmethod
    def _is_low(series: MeterSeries, reading: ReadingInput) -> bool:
        expected = series.expected_kwh(reading)
        return expected > 0 and reading.consumption_kwh < expected * MAX_RATIO_TO_EXPECTED

    @staticmethod
    def _to_signal(series: MeterSeries, low_run: list[ReadingInput], recovered_at: ReadingInput | None) -> Signal:
        expected = median(series.expected_kwh(reading) for reading in low_run)
        observed = median(reading.consumption_kwh for reading in low_run)
        change = percent_change(expected, observed)
        hours = len(low_run)

        return Signal(
            kind=SignalKind.CONSUMPTION_DROP,
            started_at=low_run[0].timestamp,
            ended_at=recovered_at.timestamp if recovered_at else None,
            magnitude=round(change, 1),
            baseline_value=round(expected, 2),
            observed_value=round(observed, 2),
            description=f"Consumption {abs(change):.1f}% below the hourly baseline for {hours} consecutive hours",
        )
