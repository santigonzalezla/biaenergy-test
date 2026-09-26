import pytest

from app.analysis.pipeline import AnomalyAnalyzer
from app.domain.enums import AnomalyType, Severity
from app.domain.models import AnalysisRequest, Explanation, Finding, MeterInput


class RecordingExplainer:
    def __init__(self):
        self.explained: list[Finding] = []

    def explain(self, finding: Finding) -> Explanation:
        self.explained.append(finding)
        return Explanation(
            reason=f"{finding.meter_code} classified by {finding.rule_id}",
            recommended_action=f"Review {finding.meter_code}",
        )


@pytest.fixture(scope="module")
def analysis(dataset_request):
    explainer = RecordingExplainer()
    return AnomalyAnalyzer(explainer).analyze(dataset_request), explainer


def test_findings_are_ranked_by_priority(analysis):
    result, _ = analysis

    assert [finding.meter_code for finding in result.findings] == ["M-109", "M-112", "M-104", "M-106"]
    assert result.meters_analyzed == 12


@pytest.mark.parametrize(
    ("code", "anomaly_type", "severity"),
    [
        pytest.param("M-109", AnomalyType.REAL_ANOMALY, Severity.HIGH, id="M-109"),
        pytest.param("M-104", AnomalyType.EXPLAINABLE_ANOMALY, Severity.MEDIUM, id="M-104"),
        pytest.param("M-106", AnomalyType.FALSE_POSITIVE, Severity.LOW, id="M-106"),
        pytest.param("M-112", AnomalyType.DATA_QUALITY, Severity.HIGH, id="M-112"),
    ],
)
def test_expected_outcome_per_case(analysis, code, anomaly_type, severity):
    result, _ = analysis
    [finding] = [finding for finding in result.findings if finding.meter_code == code]

    assert finding.type == anomaly_type
    assert finding.severity == severity


def test_m109_finding_carries_the_full_evidence(analysis):
    result, _ = analysis
    finding = result.findings[0]

    assert finding.variation_pct > 100
    assert finding.current_kwh > finding.baseline_kwh
    assert finding.window_end is None
    assert finding.related_event_id is not None
    assert {variable.name for variable in finding.changed_variables} == {"consumption_kwh", "power_factor", "current"}


def test_m112_consumption_is_stable_while_voltage_is_not(analysis):
    result, _ = analysis
    [finding] = [finding for finding in result.findings if finding.meter_code == "M-112"]

    assert abs(finding.variation_pct) < 5
    assert {"voltage", "voltage_step"} <= {variable.name for variable in finding.changed_variables}


def test_every_finding_is_explained_once(analysis):
    result, explainer = analysis

    assert sorted(finding.meter_code for finding in explainer.explained) == ["M-104", "M-106", "M-109", "M-112"]
    assert all(finding.reason and finding.recommended_action for finding in result.findings)


def test_meters_without_readings_are_counted_but_not_analyzed():
    request = AnalysisRequest(meters=[MeterInput(id="empty", code="M-999")], readings=[])

    result = AnomalyAnalyzer(RecordingExplainer()).analyze(request)

    assert result.meters_analyzed == 1
    assert result.findings == []
