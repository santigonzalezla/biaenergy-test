from datetime import UTC, datetime

import pytest

from app.analysis.pipeline import AnomalyAnalyzer
from app.domain.enums import AnomalyType, Severity
from app.domain.models import Finding
from app.explanation.template import TemplateExplainer, number, percent


@pytest.fixture(scope="module")
def findings_by_code(dataset_request):
    result = AnomalyAnalyzer(TemplateExplainer()).analyze(dataset_request)
    return {finding.meter_code: finding for finding in result.findings}


@pytest.mark.parametrize(
    ("code", "reason_fragments", "action_fragment"),
    [
        pytest.param(
            "M-109",
            ["110,5%", "12-sep a las 14:00", "No operational event reported", "factor de potencia cayó de 0,94 a 0,74"],
            "Inspeccionar con prioridad",
            id="real anomaly cites the missing cause and the electrical evidence",
        ),
        pytest.param(
            "M-112",
            ["241,2 V", "25,5 V", "estable", "falla de medición", "Intermittent readings"],
            "revisar el medidor M-112",
            id="data quality points to the meter, not the plant",
        ),
        pytest.param(
            "M-104",
            ["46,5%", "New production line activated", "coherente con la nueva carga"],
            "actualizar el baseline",
            id="explainable anomaly asks to validate and update the baseline",
        ),
        pytest.param(
            "M-106",
            ["12 horas", "se recuperó", "Scheduled maintenance outage", "falso positivo"],
            "No requiere acción",
            id="false positive requires no action",
        ),
    ],
)
def test_explanations_cite_the_evidence(findings_by_code, code, reason_fragments, action_fragment):
    finding = findings_by_code[code]

    for fragment in reason_fragments:
        assert fragment in finding.reason
    assert action_fragment in finding.recommended_action


def synthetic_finding(rule_id: str) -> Finding:
    return Finding(
        meter_id="m",
        meter_code="M-900",
        type=AnomalyType.REAL_ANOMALY,
        severity=Severity.MEDIUM,
        rule_id=rule_id,
        confidence=0.6,
        priority_score=30,
        detected_at=datetime(2026, 9, 12, 19, tzinfo=UTC),
        window_start=datetime(2026, 9, 12, 19, tzinfo=UTC),
        window_end=None,
        baseline_kwh=40,
        current_kwh=20,
        variation_pct=-50,
        changed_variables=[],
        signals=[],
        related_event=None,
    )


def test_dates_are_shown_in_the_site_timezone():
    explanation = TemplateExplainer("America/Bogota").explain(synthetic_finding("R5_UNEXPLAINED_DROP"))

    assert "12-sep a las 14:00" in explanation.reason


def test_unknown_rules_fall_back_to_a_generic_explanation():
    explanation = TemplateExplainer().explain(synthetic_finding("R99_FUTURE_RULE"))

    assert "M-900" in explanation.reason
    assert explanation.recommended_action


@pytest.mark.parametrize(
    ("value", "signed", "expected"),
    [
        pytest.param(110.5, False, "110,5%", id="decimal comma"),
        pytest.param(0.46, True, "+0,5%", id="explicit sign for increases"),
        pytest.param(-3.2, True, "-3,2%", id="negative keeps its sign"),
    ],
)
def test_percent_format(value, signed, expected):
    assert percent(value, signed=signed) == expected


def test_number_format():
    assert number(485.11, 0) == "485"
