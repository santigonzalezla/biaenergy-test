import {screen, waitFor, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {POLL_INTERVAL_MS} from '@/context/analysis.ts';
import {clearSession} from '@/lib/session.ts';
import {analysis, anomalyList, meterPage, signIn, summary} from '@/test/fixtures.ts';
import {apiError, mockApi, ok} from '@/test/mockApi.ts';
import {renderApp} from '@/test/renderApp.tsx';

const dashboardRoutes = {
    'GET /dashboard/summary': ok(summary()),
    'GET /anomalies': ok(anomalyList()),
    'GET /meters?limit=100&sortBy=code': ok(meterPage()),
};

describe('DashboardPage', () =>
{
    beforeEach(() =>
    {
        clearSession();
        signIn();
    });

    it('shows the KPIs, the ranking by priority and the meters on alert', async () =>
    {
        mockApi({...dashboardRoutes, 'GET /ai/analysis/latest': ok(analysis())});

        renderApp('/dashboard');

        expect(await screen.findByText('155.250,85')).toBeInTheDocument();
        expect(screen.getByText('4.032')).toBeInTheDocument();
        expect(screen.getByText('2 con alerta')).toBeInTheDocument();
        expect(screen.getAllByText('2 de alta prioridad')).toHaveLength(2);

        const priority = within(screen.getByText('Prioridad de atención').closest('section')!);
        await priority.findByText('M-109');
        const items = priority.getAllByRole('listitem');
        expect(items[0]).toHaveTextContent('M-109');
        expect(items[0]).toHaveTextContent('Anomalía real');
        expect(items[items.length - 1]).toHaveTextContent('Falso positivo');

        const alerting = within(screen.getByText('Medidores con alerta').closest('section')!);
        expect(alerting.getByText('M-109')).toBeInTheDocument();
        expect(alerting.getByText('Crítico')).toBeInTheDocument();
        expect(alerting.getByText('Alerta')).toBeInTheDocument();
        expect(alerting.queryByText('M-104')).not.toBeInTheDocument();

        expect(await screen.findByText('Último análisis IA')).toBeInTheDocument();
        expect(screen.getByText('12 medidores analizados')).toBeInTheDocument();
    });

    it('invites to run the first analysis when there is none', async () =>
    {
        mockApi({
            ...dashboardRoutes,
            'GET /anomalies': ok({analysisId: null, data: []}),
            'GET /ai/analysis/latest': apiError(404, 'NO_ANALYSIS_YET'),
        });

        renderApp('/dashboard');

        expect(await screen.findByText('Aún no hay análisis')).toBeInTheDocument();
        expect(await screen.findByText('Sin anomalías')).toBeInTheDocument();
    });

    it('shows a retryable error when the summary fails', async () =>
    {
        mockApi({
            ...dashboardRoutes,
            'GET /dashboard/summary': [apiError(500, 'INTERNAL_ERROR'), ok(summary())],
            'GET /ai/analysis/latest': ok(analysis()),
        });
        const user = userEvent.setup();

        renderApp('/dashboard');

        expect(await screen.findByText('No se pudo cargar el resumen')).toBeInTheDocument();
        await user.click(screen.getByRole('button', {name: 'Reintentar'}));

        expect(await screen.findByText('155.250,85')).toBeInTheDocument();
    });

    it('runs an analysis, follows its progress and refreshes the dashboard when it completes', async () =>
    {
        const running = analysis({status: 'RUNNING', currentStep: 'analyzing', progress: 40, finishedAt: null, metersAnalyzed: 0});
        const fetchMock = mockApi({
            ...dashboardRoutes,
            'GET /ai/analysis/latest': apiError(404, 'NO_ANALYSIS_YET'),
            'POST /ai/analyze': ok(analysis({status: 'PENDING', currentStep: null, progress: 0, finishedAt: null}), 202),
            [`GET /ai/analysis/${running.id}`]: [ok(running), ok(analysis())],
        });
        const user = userEvent.setup();

        renderApp('/dashboard');

        await screen.findByText('Aún no hay análisis');
        await user.click(screen.getAllByRole('button', {name: 'Ejecutar análisis IA'})[0]);

        const drawer = screen.getByRole('dialog', {name: 'Progreso del análisis IA'});

        expect(await within(drawer).findByText('40 %', {}, {timeout: POLL_INTERVAL_MS * 2})).toBeInTheDocument();
        expect(within(drawer).getByText('Analizando con IA')).toBeInTheDocument();

        expect(await within(drawer).findByRole('button', {name: 'Ver anomalías'}, {timeout: POLL_INTERVAL_MS * 2})).toBeInTheDocument();
        expect(within(drawer).getByText('Alta prioridad').nextSibling).toHaveTextContent('2');

        await waitFor(() =>
        {
            const summaryCalls = fetchMock.mock.calls.filter(([url]) => String(url).endsWith('/dashboard/summary'));
            expect(summaryCalls).toHaveLength(2);
        });
    });

    it('shows the failure and lets the user retry', async () =>
    {
        const pending = analysis({status: 'PENDING', currentStep: null, progress: 0, finishedAt: null});
        mockApi({
            ...dashboardRoutes,
            'GET /ai/analysis/latest': ok(pending),
            [`GET /ai/analysis/${pending.id}`]: ok(analysis({status: 'FAILED', currentStep: 'analyzing', progress: 40, error: 'ai service timed out'})),
        });
        const user = userEvent.setup();

        renderApp('/dashboard');

        await user.click(await screen.findByRole('button', {name: 'Ver progreso'}));

        const drawer = screen.getByRole('dialog', {name: 'Progreso del análisis IA'});
        expect(await within(drawer).findByRole('alert', {}, {timeout: POLL_INTERVAL_MS * 2})).toHaveTextContent('ai service timed out');
        expect(within(drawer).getByRole('button', {name: 'Reintentar'})).toBeInTheDocument();
    });
});
