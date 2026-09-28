import type {Session} from '@/interfaces/auth.ts';
import type {Analysis, AnomalyList, DashboardSummary, Meter, Page} from '@/interfaces/interfaces.ts';
import {saveSession} from '@/lib/session.ts';

export const ADMIN = {id: 'f90768ae-b4ed-425e-bdbd-918714c97484', email: 'admin@bia.app', name: 'Admin BIA'};

export const signIn = () =>
{
    const session: Session = {
        token: 'valid.jwt.token',
        expiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
        user: ADMIN,
    };

    saveSession(session);

    return session;
}

export const analysis = (overrides: Partial<Analysis> = {}): Analysis => ({
    id: '7e91463d-77f7-42c3-a314-b205da8c31a9',
    numId: 5,
    status: 'COMPLETED',
    currentStep: 'completed',
    progress: 100,
    metersAnalyzed: 12,
    anomalies: 4,
    highPriority: 2,
    error: null,
    startedAt: '2026-09-27T22:13:08Z',
    finishedAt: '2026-09-27T22:13:30Z',
    createdAt: '2026-09-27T22:13:08Z',
    ...overrides,
});

export const summary = (overrides: Partial<DashboardSummary> = {}): DashboardSummary => ({
    metersCount: 12,
    metersWithAlerts: 2,
    readingsCount: 4032,
    totalKwh: 155250.85,
    openAnomalies: 4,
    highPriorityAnomalies: 2,
    lastAnalysis: null,
    ...overrides,
});

export const anomalyList = (): AnomalyList => ({
    analysisId: analysis().id,
    data: [
        {
            id: 'a-109', numId: 1, meterId: 'm-109', meterCode: 'M-109', meterName: 'Medidor M-109',
            type: 'REAL_ANOMALY', severity: 'HIGH', status: 'OPEN', ruleId: 'R4_REAL_ANOMALY',
            confidence: 0.9, priorityScore: 90, variationPct: 110.5, detectedAt: '2026-09-12T19:00:00Z',
            reason: 'Aumento sostenido del consumo del 110,5 % sin evento que lo explique.',
        },
        {
            id: 'a-112', numId: 2, meterId: 'm-112', meterCode: 'M-112', meterName: 'Medidor M-112',
            type: 'DATA_QUALITY', severity: 'HIGH', status: 'OPEN', ruleId: 'R3_DATA_QUALITY',
            confidence: 0.9, priorityScore: 31.5, variationPct: 0.5, detectedAt: '2026-09-13T05:00:00Z',
            reason: 'Consumo estable con voltaje y factor de potencia inconsistentes.',
        },
        {
            id: 'a-106', numId: 4, meterId: 'm-106', meterCode: 'M-106', meterName: 'Medidor M-106',
            type: 'FALSE_POSITIVE', severity: 'LOW', status: 'OPEN', ruleId: 'R2_SCHEDULED_OUTAGE',
            confidence: 0.8, priorityScore: 0.6, variationPct: -79.8, detectedAt: '2026-09-08T05:00:00Z',
            reason: 'Caída explicada por un corte programado.',
        },
    ],
});

const meter = (code: string, status: Meter['status'], variationPct: number): Meter => ({
    id: `m-${code.slice(2)}`,
    numId: Number(code.slice(2)),
    code,
    name: `Medidor ${code}`,
    location: '',
    sector: 'INDUSTRIAL',
    nominalVoltage: 220,
    maxCurrent: null,
    contractedPowerKw: null,
    status,
    createdAt: '2026-09-25T06:08:53Z',
    updatedAt: '2026-09-27T22:13:30Z',
    stats: {baselineKwh: 43.7, recentKwh: 43.7 * (1 + variationPct / 100), variationPct, minPowerFactor: 0.9, totalKwh: 17000, lastReadingAt: '2026-09-15T04:00:00Z'},
});

export const meterPage = (): Page<Meter> => ({
    data: [meter('M-101', 'OK', -0.5), meter('M-104', 'OK', 47.2), meter('M-109', 'CRITICAL', 109.8), meter('M-112', 'ALERT', 0.1)],
    total: 4,
    page: 1,
    limit: 100,
});
