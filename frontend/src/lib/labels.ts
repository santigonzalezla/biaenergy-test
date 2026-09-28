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

export const EVENT_TYPE: Record<EventType, string> = {
    OPERATIONAL_CHANGE: 'Cambio operativo',
    SCHEDULED_OUTAGE: 'Corte programado',
    DATA_QUALITY: 'Calidad de datos',
    MAINTENANCE: 'Mantenimiento',
    UNKNOWN: 'Desconocido',
};
