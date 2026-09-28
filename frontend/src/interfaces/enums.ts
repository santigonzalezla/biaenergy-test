export const MeterStatus = {
    OK: 'OK',
    ALERT: 'ALERT',
    CRITICAL: 'CRITICAL',
} as const;

export type MeterStatus = typeof MeterStatus[keyof typeof MeterStatus];

export const AnomalyType = {
    REAL_ANOMALY: 'REAL_ANOMALY',
    EXPLAINABLE_ANOMALY: 'EXPLAINABLE_ANOMALY',
    FALSE_POSITIVE: 'FALSE_POSITIVE',
    DATA_QUALITY: 'DATA_QUALITY',
} as const;

export type AnomalyType = typeof AnomalyType[keyof typeof AnomalyType];

export const AnomalySeverity = {
    LOW: 'LOW',
    MEDIUM: 'MEDIUM',
    HIGH: 'HIGH',
} as const;

export type AnomalySeverity = typeof AnomalySeverity[keyof typeof AnomalySeverity];

export const AnomalyStatus = {
    OPEN: 'OPEN',
    INVESTIGATING: 'INVESTIGATING',
    RESOLVED: 'RESOLVED',
    DISMISSED: 'DISMISSED',
} as const;

export type AnomalyStatus = typeof AnomalyStatus[keyof typeof AnomalyStatus];

export const AnalysisStatus = {
    PENDING: 'PENDING',
    RUNNING: 'RUNNING',
    COMPLETED: 'COMPLETED',
    FAILED: 'FAILED',
} as const;

export type AnalysisStatus = typeof AnalysisStatus[keyof typeof AnalysisStatus];

export const AnalysisStep = {
    LOADING_DATA: 'loading_data',
    ANALYZING: 'analyzing',
    SAVING: 'saving',
    COMPLETED: 'completed',
} as const;

export type AnalysisStep = typeof AnalysisStep[keyof typeof AnalysisStep];

export const EventType = {
    OPERATIONAL_CHANGE: 'OPERATIONAL_CHANGE',
    SCHEDULED_OUTAGE: 'SCHEDULED_OUTAGE',
    DATA_QUALITY: 'DATA_QUALITY',
    MAINTENANCE: 'MAINTENANCE',
    UNKNOWN: 'UNKNOWN',
} as const;

export type EventType = typeof EventType[keyof typeof EventType];

export const MeterSortField = {
    CODE: 'code',
    NAME: 'name',
    STATUS: 'status',
} as const;

export type MeterSortField = typeof MeterSortField[keyof typeof MeterSortField];

export const SortDirection = {
    ASC: 'asc',
    DESC: 'desc',
} as const;

export type SortDirection = typeof SortDirection[keyof typeof SortDirection];

export const ImportKind = {
    READINGS: 'READINGS',
    EVENTS: 'EVENTS',
} as const;

export type ImportKind = typeof ImportKind[keyof typeof ImportKind];

export const ImportStatus = {
    COMPLETED: 'COMPLETED',
    FAILED: 'FAILED',
} as const;

export type ImportStatus = typeof ImportStatus[keyof typeof ImportStatus];
