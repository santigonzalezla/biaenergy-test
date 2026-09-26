from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel

from app.domain.enums import AnomalyType, EventType, Severity, SignalKind


class ApiModel(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True, frozen=True)


# Input
class MeterInput(ApiModel):
    id: str
    code: str
    nominal_voltage: float = Field(default=220, gt=0)
    max_current: float | None = Field(default=None, gt=0)


class ReadingInput(ApiModel):
    meter_id: str
    timestamp: datetime
    consumption_kwh: float = Field(ge=0)
    voltage: float = Field(ge=0)
    current: float = Field(ge=0)
    power_factor: float = Field(ge=0, le=1)


class EventInput(ApiModel):
    id: str
    meter_id: str
    timestamp: datetime
    type: EventType
    description: str = ""


class AnalysisRequest(ApiModel):
    timezone: str = "America/Bogota"
    meters: list[MeterInput]
    readings: list[ReadingInput]
    events: list[EventInput] = []


# Output
class Signal(ApiModel):
    kind: SignalKind
    detected_at: datetime
    magnitude: float
    baseline_value: float
    observed_value: float
    description: str


class ChangedVariable(ApiModel):
    name: str
    baseline: float
    observed: float
    change_pct: float


class Finding(ApiModel):
    meter_id: str
    meter_code: str
    type: AnomalyType
    severity: Severity
    rule_id: str
    confidence: float = Field(ge=0, le=1)
    priority_score: float = Field(ge=0)
    detected_at: datetime
    window_start: datetime
    window_end: datetime | None
    baseline_kwh: float
    current_kwh: float
    variation_pct: float
    changed_variables: list[ChangedVariable]
    signals: list[Signal]
    related_event_id: str | None
    reason: str
    recommended_action: str


class AnalysisResult(ApiModel):
    meters_analyzed: int
    findings: list[Finding]
