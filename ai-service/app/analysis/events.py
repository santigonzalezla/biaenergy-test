import re
from collections.abc import Sequence
from dataclasses import dataclass
from datetime import timedelta

from app.domain.enums import EventType, SignalKind
from app.domain.models import EventInput, Signal

CORRELATION_WINDOW = timedelta(hours=6)
DURATION_TOLERANCE = timedelta(hours=1)
DURATION_IN_TEXT = re.compile(r"(\d+)\s*hours?", re.IGNORECASE)

EXPLAINING_EVENTS: dict[SignalKind, set[EventType]] = {
    SignalKind.CONSUMPTION_SURGE: {EventType.OPERATIONAL_CHANGE},
    SignalKind.CONSUMPTION_DROP: {EventType.SCHEDULED_OUTAGE, EventType.MAINTENANCE},
}


@dataclass(frozen=True)
class EventCorrelation:
    event: EventInput
    explains: bool
    duration_matches: bool | None


def correlate(signal: Signal, meter_events: Sequence[EventInput]) -> EventCorrelation | None:
    nearby = [event for event in meter_events if abs(event.timestamp - signal.started_at) <= CORRELATION_WINDOW]
    if not nearby:
        return None

    event = min(
        nearby,
        key=lambda candidate: (not explains(signal, candidate), abs(candidate.timestamp - signal.started_at)),
    )

    return EventCorrelation(
        event=event,
        explains=explains(signal, event),
        duration_matches=duration_matches(signal, event),
    )


def explains(signal: Signal, event: EventInput) -> bool:
    return event.type in EXPLAINING_EVENTS.get(signal.kind, set())


def duration_matches(signal: Signal, event: EventInput) -> bool | None:
    announced = DURATION_IN_TEXT.search(event.description)
    if announced is None or signal.ended_at is None:
        return None

    announced_duration = timedelta(hours=int(announced.group(1)))
    observed_duration = signal.ended_at - signal.started_at
    return abs(observed_duration - announced_duration) <= DURATION_TOLERANCE
