import type {AnalysisStatus, AnalysisStep, AnomalySeverity, AnomalyStatus, AnomalyType, EventType, MeterStatus} from '@/interfaces/enums.ts';

export type Tone = 'real' | 'explainable' | 'falsePositive' | 'dataQuality' | 'accent' | 'neutral';

export interface LabelWithTone {
    label: string;
    tone: Tone;
}

export const ANOMALY_TYPE: Record<AnomalyType, LabelWithTone> = {
    REAL_ANOMALY: {label: 'Anomalía real', tone: 'real'},
    EXPLAINABLE_ANOMALY: {label: 'Explicable', tone: 'explainable'},
    FALSE_POSITIVE: {label: 'Falso positivo', tone: 'falsePositive'},
    DATA_QUALITY: {label: 'Calidad de datos', tone: 'dataQuality'},
};

export const SEVERITY: Record<AnomalySeverity, LabelWithTone> = {
    HIGH: {label: 'Alta', tone: 'real'},
    MEDIUM: {label: 'Media', tone: 'explainable'},
    LOW: {label: 'Baja', tone: 'neutral'},
};

export const ANOMALY_STATUS: Record<AnomalyStatus, LabelWithTone> = {
    OPEN: {label: 'Abierta', tone: 'real'},
    INVESTIGATING: {label: 'En investigación', tone: 'explainable'},
    RESOLVED: {label: 'Resuelta', tone: 'accent'},
    DISMISSED: {label: 'Descartada', tone: 'neutral'},
};

export const METER_STATUS: Record<MeterStatus, LabelWithTone> = {
    OK: {label: 'Normal', tone: 'accent'},
    ALERT: {label: 'Alerta', tone: 'explainable'},
    CRITICAL: {label: 'Crítico', tone: 'real'},
};

export const ANALYSIS_STATUS: Record<AnalysisStatus, LabelWithTone> = {
    PENDING: {label: 'En cola', tone: 'neutral'},
    RUNNING: {label: 'En progreso', tone: 'accent'},
    COMPLETED: {label: 'Completado', tone: 'accent'},
    FAILED: {label: 'Fallido', tone: 'real'},
};

export const ANALYSIS_STEP: Record<AnalysisStep, string> = {
    loading_data: 'Cargando datos',
    analyzing: 'Analizando con IA',
    saving: 'Guardando resultados',
    completed: 'Completado',
};

export const ALLOWED_TRANSITIONS: Record<AnomalyStatus, AnomalyStatus[]> = {
    OPEN: ['INVESTIGATING', 'RESOLVED', 'DISMISSED'],
    INVESTIGATING: ['OPEN', 'RESOLVED', 'DISMISSED'],
    RESOLVED: ['OPEN'],
    DISMISSED: ['OPEN'],
};

export const TRANSITION_ACTION: Record<AnomalyStatus, string> = {
    OPEN: 'Reabrir',
    INVESTIGATING: 'Marcar en investigación',
    RESOLVED: 'Marcar como resuelta',
    DISMISSED: 'Descartar',
};

export const SIGNAL_KIND: Record<string, string> = {
    CONSUMPTION_SURGE: 'Aumento sostenido de consumo',
    CONSUMPTION_DROP: 'Caída sostenida de consumo',
    VOLTAGE_OUT_OF_RANGE: 'Voltaje fuera de rango (±5 %)',
    VOLTAGE_INSTABILITY: 'Inestabilidad de voltaje',
    LOW_POWER_FACTOR: 'Factor de potencia bajo (< 0,80)',
    OVERCURRENT: 'Sobrecorriente (> 1,5 × pico previo)',
};

export interface VariableLabel {
    label: string;
    unit: string;
    digits: number;
}

export const VARIABLE: Record<string, VariableLabel> = {
    consumption_kwh: {label: 'Consumo', unit: 'kWh/h', digits: 1},
    voltage: {label: 'Voltaje', unit: 'V', digits: 1},
    voltage_step: {label: 'Salto de voltaje', unit: 'V', digits: 1},
    power_factor: {label: 'Factor de potencia', unit: '', digits: 2},
    current: {label: 'Corriente', unit: 'A', digits: 0},
};

export const variableOf = (name: string): VariableLabel => VARIABLE[name] ?? {label: name, unit: '', digits: 2};

export const EVENT_TYPE: Record<EventType, string> = {
    OPERATIONAL_CHANGE: 'Cambio operativo',
    SCHEDULED_OUTAGE: 'Corte programado',
    DATA_QUALITY: 'Calidad de datos',
    MAINTENANCE: 'Mantenimiento',
    UNKNOWN: 'Desconocido',
};
