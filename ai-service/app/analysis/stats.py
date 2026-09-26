from collections.abc import Sequence
from statistics import median

MAD_TO_STD = 1.4826


def median_absolute_deviation(values: Sequence[float]) -> float:
    center = median(values)
    return median(abs(value - center) for value in values)


def robust_z_score(value: float, center: float, mad: float) -> float:
    if mad == 0:
        return 0.0
    return (value - center) / (MAD_TO_STD * mad)


def percent_change(baseline: float, observed: float) -> float:
    if baseline == 0:
        return 0.0
    return (observed - baseline) / baseline * 100
