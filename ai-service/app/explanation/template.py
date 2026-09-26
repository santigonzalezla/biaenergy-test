from collections.abc import Callable
from datetime import datetime
from zoneinfo import ZoneInfo

from app.domain.enums import EventType
from app.domain.models import ChangedVariable, Explanation, Finding

MONTHS = ("ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic")


class TemplateExplainer:
    def __init__(self, timezone: str = "America/Bogota"):
        self._timezone = ZoneInfo(timezone)
        self._builders: dict[str, Callable[[Finding], Explanation]] = {
            "R1_MEASUREMENT_FAULT": self._measurement_fault,
            "R2_SCHEDULED_OUTAGE": self._scheduled_outage,
            "R3_EXPLAINED_SURGE": self._explained_surge,
            "R4_UNEXPLAINED_SURGE": self._unexplained_surge,
            "R5_UNEXPLAINED_DROP": self._unexplained_drop,
            "R6_EQUIPMENT_FAULT": self._equipment_fault,
        }

    def explain(self, finding: Finding) -> Explanation:
        return self._builders.get(finding.rule_id, self._generic)(finding)

    def _unexplained_surge(self, finding: Finding) -> Explanation:
        reason = (
            f"El consumo de {finding.meter_code} aumentó un {percent(finding.variation_pct)} "
            f"(de {number(finding.baseline_kwh)} a {number(finding.current_kwh)} kWh por hora) "
            f"desde el {self._date(finding.window_start)} {self._still_active(finding)}. "
            f"{self._missing_cause(finding)}"
        )

        equipment = self._equipment_evidence(finding)
        if equipment:
            reason += (
                f" Además, {equipment}: la instalación está trabajando en condiciones anormales, "
                "no solo produciendo más."
            )

        action = (
            f"Inspeccionar con prioridad los equipos asociados a {finding.meter_code} (motores, compresores y banco "
            "de condensadores) para descartar sobrecarga o falla. Confirmar con operación si hubo un cambio no "
            "registrado; si no lo hubo, programar mantenimiento correctivo."
        )
        return Explanation(reason=reason, recommended_action=action)

    def _explained_surge(self, finding: Finding) -> Explanation:
        reason = (
            f"El consumo de {finding.meter_code} aumentó un {percent(finding.variation_pct)} "
            f"(de {number(finding.baseline_kwh)} a {number(finding.current_kwh)} kWh por hora) "
            f"desde el {self._date(finding.window_start)}. {self._event_sentence(finding)} "
            "El comportamiento eléctrico se mantiene normal, por lo que el aumento es coherente con la nueva carga."
        )
        action = (
            "Validar con operación que el nuevo nivel de consumo corresponde a lo esperado para ese cambio y, "
            f"si es así, actualizar el baseline de {finding.meter_code} para que se considere normal."
        )
        return Explanation(reason=reason, recommended_action=action)

    def _scheduled_outage(self, finding: Finding) -> Explanation:
        hours = self._duration_hours(finding)
        reason = (
            f"El consumo de {finding.meter_code} cayó un {percent(abs(finding.variation_pct))} durante {hours} horas "
            f"(del {self._date(finding.window_start)} al {self._date(finding.window_end)}) y luego se recuperó. "
            f"{self._event_sentence(finding)} La caída corresponde a ese evento planificado: es un falso positivo."
        )
        action = (
            "No requiere acción correctiva. Registrar el período como mantenimiento programado para excluirlo "
            "de los indicadores de desempeño."
        )
        return Explanation(reason=reason, recommended_action=action)

    def _measurement_fault(self, finding: Finding) -> Explanation:
        reason = (
            f"Las lecturas eléctricas de {finding.meter_code} no son confiables desde el "
            f"{self._date(finding.window_start)}: {self._voltage_evidence(finding)}, mientras el consumo se mantiene "
            f"estable ({percent(finding.variation_pct, signed=True)}). Esa combinación es físicamente incoherente e "
            "indica una falla de medición, no de la instalación."
        )
        if finding.related_event is not None and finding.related_event.type == EventType.DATA_QUALITY:
            reason += f' Hay un registro que lo confirma: "{finding.related_event.description}".'

        action = (
            f"Enviar un técnico a revisar el medidor {finding.meter_code} (conexiones, transformadores de medida y "
            f"sensor) y marcar sus lecturas desde el {self._date(finding.window_start)} como no confiables hasta "
            "su verificación."
        )
        return Explanation(reason=reason, recommended_action=action)

    def _unexplained_drop(self, finding: Finding) -> Explanation:
        reason = (
            f"El consumo de {finding.meter_code} cayó un {percent(abs(finding.variation_pct))} desde el "
            f"{self._date(finding.window_start)} {self._still_active(finding)} sin un evento planificado que "
            "lo explique, o durante más tiempo del anunciado."
        )
        action = (
            f"Verificar en sitio si los equipos de {finding.meter_code} están detenidos o fallando, y confirmar con "
            "operación si hubo un corte no registrado."
        )
        return Explanation(reason=reason, recommended_action=action)

    def _equipment_fault(self, finding: Finding) -> Explanation:
        reason = (
            f"En {finding.meter_code}, {self._equipment_evidence(finding)} desde el "
            f"{self._date(finding.window_start)}, sin un cambio significativo de consumo."
        )
        action = (
            f"Revisar el banco de condensadores y el estado de los motores de {finding.meter_code}; un factor de "
            "potencia bajo genera recargos por energía reactiva."
        )
        return Explanation(reason=reason, recommended_action=action)

    def _generic(self, finding: Finding) -> Explanation:
        since = self._date(finding.window_start)
        return Explanation(
            reason=f"{finding.meter_code} presenta un comportamiento anómalo desde el {since}.",
            recommended_action=f"Revisar el medidor {finding.meter_code} y sus equipos asociados.",
        )

    # Sentences

    def _missing_cause(self, finding: Finding) -> str:
        event = finding.related_event
        if event is not None and event.type == EventType.UNKNOWN:
            return f'No hay un evento operativo que lo justifique (registro: "{event.description}").'
        return "No hay un evento operativo registrado que lo justifique."

    def _event_sentence(self, finding: Finding) -> str:
        event = finding.related_event
        if event is None:
            return ""
        return f'Coincide con el evento registrado el {self._date(event.timestamp)}: "{event.description}".'

    def _still_active(self, finding: Finding) -> str:
        return "y se mantiene" if finding.window_end is None else f"hasta el {self._date(finding.window_end)}"

    @staticmethod
    def _equipment_evidence(finding: Finding) -> str:
        parts = []
        power_factor = changed(finding, "power_factor")
        if power_factor:
            parts.append(
                f"el factor de potencia cayó de {number(power_factor.baseline)} a {number(power_factor.observed)}"
            )
        current = changed(finding, "current")
        if current:
            parts.append(f"la corriente subió de {number(current.baseline, 0)} a {number(current.observed, 0)} A")
        return " y ".join(parts)

    @staticmethod
    def _voltage_evidence(finding: Finding) -> str:
        parts = []
        voltage = changed(finding, "voltage")
        if voltage:
            parts.append(
                f"el voltaje llegó a {number(voltage.observed, 1)} V (nominal {number(voltage.baseline, 0)} V, "
                f"fuera de la tolerancia de ±5%)"
            )
        step = changed(finding, "voltage_step")
        if step:
            parts.append(
                f"con saltos de hasta {number(step.observed, 1)} V entre horas "
                f"(lo típico es {number(step.baseline, 1)} V)"
            )
        return " ".join(parts) or "las variables eléctricas muestran valores anómalos"

    @staticmethod
    def _duration_hours(finding: Finding) -> int:
        return round((finding.window_end - finding.window_start).total_seconds() / 3600)

    def _date(self, value: datetime | None) -> str:
        if value is None:
            return "momento actual"
        local = value.astimezone(self._timezone)
        return f"{local.day}-{MONTHS[local.month - 1]} a las {local:%H:%M}"


def changed(finding: Finding, name: str) -> ChangedVariable | None:
    return next((variable for variable in finding.changed_variables if variable.name == name), None)


def number(value: float, decimals: int = 2) -> str:
    return f"{value:.{decimals}f}".replace(".", ",")


def percent(value: float, signed: bool = False) -> str:
    sign = "+" if signed and value > 0 else ""
    return f"{sign}{number(value, 1)}%"
