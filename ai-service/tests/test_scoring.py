import pytest

from app.analysis.detectors.consumption_drop import ConsumptionDropDetector
from app.analysis.detectors.level_shift import LevelShiftDetector
from app.analysis.detectors.overcurrent import OvercurrentDetector
from app.analysis.detectors.power_factor import PowerFactorDetector
from app.analysis.detectors.voltage import VoltageDetector
from app.analysis.events import correlate
from app.analysis.rules import Evidence, classify
from app.analysis.scoring import score

DETECTORS = [
    LevelShiftDetector(),
    ConsumptionDropDetector(),
    VoltageDetector(),
    PowerFactorDetector(),
    OvercurrentDetector(),
]
CASES = ["M-109", "M-112", "M-104", "M-106"]


@pytest.fixture(scope="module")
def scores(series_by_code, dataset_request):
    result = {}
    for code in CASES:
        events = [event for event in dataset_request.events if event.meter_id == code]
        signals = [signal for detector in DETECTORS for signal in detector.detect(series_by_code[code])]
        [classification] = classify([Evidence(signal, correlate(signal, events)) for signal in signals])
        result[code] = score(classification)
    return result


def test_m109_is_the_top_priority(scores):
    ranking = sorted(scores, key=lambda code: scores[code].priority, reverse=True)

    assert ranking == ["M-109", "M-112", "M-104", "M-106"]


def test_m109_leads_by_a_wide_margin(scores):
    assert scores["M-109"].priority > 2 * scores["M-112"].priority


def test_false_positive_has_negligible_priority(scores):
    assert scores["M-106"].priority < 1


@pytest.mark.parametrize(
    ("code", "expected_confidence"),
    [
        pytest.param("M-109", 0.90, id="three signals and an explicit unknown event"),
        pytest.param("M-112", 0.90, id="three signals and a data quality event"),
        pytest.param("M-106", 0.80, id="one signal, explained, duration confirmed"),
        pytest.param("M-104", 0.70, id="one signal, explained"),
    ],
)
def test_confidence_reflects_corroboration_and_context(scores, code, expected_confidence):
    assert scores[code].confidence == pytest.approx(expected_confidence)


def test_confidence_and_priority_are_independent(scores):
    assert scores["M-106"].confidence > scores["M-104"].confidence
    assert scores["M-106"].priority < scores["M-104"].priority
