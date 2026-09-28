import type {
    AnalysisStatus,
    AnalysisStep,
    AnomalySeverity,
    AnomalyStatus,
    AnomalyType,
    EventType,
    ImportKind,
    ImportStatus,
    MeterSortField,
    MeterStatus,
    SortDirection,
} from '@/interfaces/enums.ts';

export type IsoDateTime = string;

// ─── API ───

export interface ApiErrorBody {
    error: {
        code: string;
        message: string;
        details?: Record<string, string> | RowError[];
        path: string;
        timestamp: IsoDateTime;
    };
}

export interface Page<T> {
    data: T[];
    total: number;
    page: number;
    limit: number;
}

export interface Series<T> {
    from: IsoDateTime;
    to: IsoDateTime;
    data: T[];
}

// ─── Medidores ───

export interface MeterStats {
    baselineKwh: number;
    recentKwh: number;
    variationPct: number;
    minPowerFactor: number;
    totalKwh: number;
    lastReadingAt: IsoDateTime;
}

export interface Meter {
    id: string;
    numId: number;
    code: string;
    name: string;
    location: string;
    sector: string;
    nominalVoltage: number;
    maxCurrent: number | null;
    contractedPowerKw: number | null;
    status: MeterStatus;
    createdAt: IsoDateTime;
    updatedAt: IsoDateTime;
    stats: MeterStats | null;
}

export interface MeterListQuery {
    search?: string;
    status?: MeterStatus;
    sortBy?: MeterSortField;
    sortDir?: SortDirection;
    page?: number;
    limit?: number;
}

export interface CreateMeterRequest {
    code: string;
    name: string;
    location: string;
    sector: string;
    nominalVoltage?: number;
    maxCurrent?: number;
    contractedPowerKw?: number;
}

export interface UpdateMeterRequest {
    name?: string;
    location?: string;
    sector?: string;
    nominalVoltage?: number;
    maxCurrent?: number;
    contractedPowerKw?: number;
    status?: MeterStatus;
}

export interface Reading {
    timestamp: IsoDateTime;
    consumptionKwh: number;
    voltage: number;
    current: number;
    powerFactor: number;
    status: string;
}

export interface MeterEvent {
    id: string;
    timestamp: IsoDateTime;
    type: EventType;
    description: string;
}

export interface TimeRangeQuery {
    from?: IsoDateTime;
    to?: IsoDateTime;
}

export interface HourlyValue {
    hour: number;
    avgKwh: number;
    samples: number;
}

export interface HourlyProfile {
    from: IsoDateTime;
    to: IsoDateTime;
    timezone: string;
    hours: HourlyValue[];
}

// ─── Anomalías ───

export interface AnomalySummary {
    id: string;
    numId: number;
    meterId: string;
    meterCode: string;
    meterName: string;
    type: AnomalyType;
    severity: AnomalySeverity;
    status: AnomalyStatus;
    ruleId: string;
    confidence: number;
    priorityScore: number;
    variationPct: number | null;
    detectedAt: IsoDateTime;
    reason: string;
}

export interface AnomalyList {
    analysisId: string | null;
    data: AnomalySummary[];
}

export interface AnomalyListQuery {
    analysisId?: string;
    type?: AnomalyType;
    severity?: AnomalySeverity;
    status?: AnomalyStatus;
}

export interface MeterReference {
    id: string;
    code: string;
    name: string;
    location: string;
}

export interface EventReference {
    id: string;
    type: EventType;
    timestamp: IsoDateTime;
    description: string;
}

export interface Signal {
    kind: string;
    startedAt: IsoDateTime;
    endedAt: IsoDateTime | null;
    magnitude: number;
    baselineValue: number;
    observedValue: number;
    description: string;
}

export interface ChangedVariable {
    name: string;
    baseline: number;
    observed: number;
    changePct: number;
}

export interface Evidence {
    signals: Signal[];
    changedVariables: ChangedVariable[];
    relatedEvent: EventReference | null;
}

export interface AnomalyDetail {
    id: string;
    numId: number;
    analysisId: string;
    meter: MeterReference;
    type: AnomalyType;
    severity: AnomalySeverity;
    status: AnomalyStatus;
    ruleId: string;
    confidence: number;
    priorityScore: number;
    baselineKwh: number | null;
    currentKwh: number | null;
    variationPct: number | null;
    detectedAt: IsoDateTime;
    windowStart: IsoDateTime | null;
    windowEnd: IsoDateTime | null;
    reason: string;
    recommendedAction: string;
    changedVariables: string[];
    evidence: Evidence;
    relatedEvent: EventReference | null;
    resolutionNote: string | null;
    updatedAt: IsoDateTime;
}

export interface UpdateAnomalyStatusRequest {
    status: AnomalyStatus;
    note?: string;
}

// ─── Análisis IA ───

export interface Analysis {
    id: string;
    numId: number;
    status: AnalysisStatus;
    currentStep: AnalysisStep | null;
    progress: number;
    metersAnalyzed: number;
    anomalies: number;
    highPriority: number;
    error: string | null;
    startedAt: IsoDateTime | null;
    finishedAt: IsoDateTime | null;
    createdAt: IsoDateTime;
}

export interface StartAnalysisRequest {
    meterIds?: string[];
}

// ─── Dashboard ───

export interface LastAnalysis {
    id: string;
    status: AnalysisStatus;
    currentStep: AnalysisStep | null;
    progress: number;
    anomalies: number;
    highPriority: number;
    error: string | null;
    createdAt: IsoDateTime;
    finishedAt: IsoDateTime | null;
}

export interface DashboardSummary {
    metersCount: number;
    metersWithAlerts: number;
    readingsCount: number;
    totalKwh: number;
    openAnomalies: number;
    highPriorityAnomalies: number;
    lastAnalysis: LastAnalysis | null;
}

// ─── Importación ───

export interface ImportResult {
    batchId: string;
    rows: number;
    inserted: number;
    skipped: number;
    metersCreated: number;
    metersRestored: number;
    previouslyImportedAt: IsoDateTime | null;
}

export interface ImportBatch {
    id: string;
    numId: number;
    kind: ImportKind;
    status: ImportStatus;
    fileName: string;
    fileSize: number;
    checksum: string;
    rows: number;
    inserted: number;
    skipped: number;
    metersCreated: number;
    metersRestored: number;
    errorCode: string | null;
    error: string | null;
    userName: string | null;
    createdAt: IsoDateTime;
}

export interface ImportBatchList {
    data: ImportBatch[];
}

export interface RowError {
    line: number;
    column?: string;
    message: string;
}
