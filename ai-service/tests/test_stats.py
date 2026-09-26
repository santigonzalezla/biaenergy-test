import pytest

from app.analysis.stats import median_absolute_deviation, percent_change, robust_z_score


@pytest.mark.parametrize(
    ("values", "expected"),
    [
        pytest.param([1, 2, 3, 4, 5], 1, id="symmetric values"),
        pytest.param([10, 10, 10, 10], 0, id="constant values"),
        pytest.param([43, 44, 44, 45, 110], 1, id="an outlier barely moves the MAD"),
    ],
)
def test_median_absolute_deviation(values, expected):
    assert median_absolute_deviation(values) == expected


@pytest.mark.parametrize(
    ("value", "center", "mad", "expected"),
    [
        pytest.param(44, 44, 1, 0, id="value at the center"),
        pytest.param(110, 44, 1, 44.5, id="far above the center"),
        pytest.param(40, 44, 1, -2.7, id="below the center is negative"),
        pytest.param(110, 44, 0, 0, id="zero MAD does not divide by zero"),
    ],
)
def test_robust_z_score(value, center, mad, expected):
    assert robust_z_score(value, center, mad) == pytest.approx(expected, abs=0.1)


@pytest.mark.parametrize(
    ("baseline", "observed", "expected"),
    [
        pytest.param(43.7, 91.69, 109.8, id="M-109 surge"),
        pytest.param(48.74, 71.75, 47.2, id="M-104 surge"),
        pytest.param(56.2, 8.0, -85.8, id="M-106 outage drop"),
        pytest.param(0, 10, 0, id="zero baseline does not divide by zero"),
    ],
)
def test_percent_change(baseline, observed, expected):
    assert percent_change(baseline, observed) == pytest.approx(expected, abs=0.1)
