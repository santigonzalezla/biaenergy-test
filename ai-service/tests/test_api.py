import pytest
from fastapi.testclient import TestClient

from app.config import Settings
from app.main import create_app

SETTINGS = Settings(environment="test", port=8000, timezone="America/Bogota", llm_api_key=None, llm_model="none")


@pytest.fixture(scope="module")
def client():
    return TestClient(create_app(SETTINGS), raise_server_exceptions=False)


@pytest.fixture(scope="module")
def dataset_payload(dataset_request):
    return dataset_request.model_dump(mode="json", by_alias=True)


def test_health_reports_the_active_explainer(client):
    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok", "explainer": "TemplateExplainer"}


def test_analyze_returns_ranked_findings_in_camel_case(client, dataset_payload):
    response = client.post("/analyze", json=dataset_payload)

    assert response.status_code == 200
    body = response.json()
    assert body["metersAnalyzed"] == 12
    assert [finding["meterCode"] for finding in body["findings"]] == ["M-109", "M-112", "M-104", "M-106"]

    top = body["findings"][0]
    assert top["type"] == "REAL_ANOMALY"
    assert top["ruleId"] == "R4_UNEXPLAINED_SURGE"
    assert top["priorityScore"] > 50
    assert top["relatedEvent"]["type"] == "UNKNOWN"
    assert top["reason"]
    assert top["recommendedAction"]


def invalid_payload(**overrides):
    reading = {
        "meterId": "m",
        "timestamp": "2026-09-12T14:00:00-05:00",
        "consumptionKwh": 10,
        "voltage": 220,
        "current": 50,
        "powerFactor": 0.9,
    }
    reading.update(overrides)
    return {"meters": [{"id": "m", "code": "M-900"}], "readings": [reading]}


@pytest.mark.parametrize(
    ("overrides", "field"),
    [
        pytest.param({"powerFactor": 1.4}, "readings.0.powerFactor", id="power factor above 1"),
        pytest.param({"consumptionKwh": -1}, "readings.0.consumptionKwh", id="negative consumption"),
        pytest.param({"timestamp": "2026-09-12T14:00:00"}, "readings.0.timestamp", id="timestamp without timezone"),
    ],
)
def test_invalid_payload_returns_the_api_error_format(client, overrides, field):
    response = client.post("/analyze", json=invalid_payload(**overrides))

    assert response.status_code == 400
    error = response.json()["error"]
    assert error["code"] == "VALIDATION_ERROR"
    assert error["path"] == "/analyze"
    assert field in [detail["field"] for detail in error["details"]]


@pytest.mark.parametrize(
    ("method", "path", "status", "code"),
    [
        pytest.param("get", "/nope", 404, "ROUTE_NOT_FOUND", id="unknown route"),
        pytest.param("get", "/analyze", 405, "METHOD_NOT_ALLOWED", id="wrong method"),
    ],
)
def test_http_errors_use_the_api_error_format(client, method, path, status, code):
    response = getattr(client, method)(path)

    assert response.status_code == status
    assert response.json()["error"]["code"] == code
