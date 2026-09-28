import {screen, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {clearSession} from '@/lib/session.ts';
import {analysis, anomalyList, signIn} from '@/test/fixtures.ts';
import {apiError, mockApi, ok, type MockRoutes} from '@/test/mockApi.ts';
import {renderApp} from '@/test/renderApp.tsx';

const routes = (extra: MockRoutes = {}): MockRoutes => ({
    'GET /ai/analysis/latest': ok(analysis()),
    'GET /anomalies': ok(anomalyList()),
    [`GET /ai/analysis/${analysis().id}`]: ok(analysis()),
    ...extra,
});

const cards = () => screen.getAllByRole('article');

describe('AnomaliesPage', () =>
{
    beforeEach(() =>
    {
        clearSession();
        signIn();
    });

    it('lists the anomalies by priority with the analysis they come from', async () =>
    {
        mockApi(routes());

        await renderApp('/anomalies');

        expect(await screen.findByText(/Resultado del análisis #5 · 27 de sept de 2026, 17:13/)).toBeInTheDocument();

        const [first, second, third] = cards();
        expect(within(first).getByText('#1')).toBeInTheDocument();
        expect(within(first).getByText('M-109')).toBeInTheDocument();
        expect(within(first).getByText('Anomalía real')).toBeInTheDocument();
        expect(within(first).getByText('90 %')).toBeInTheDocument();
        expect(within(first).getByText('+110,5 %')).toBeInTheDocument();
        expect(within(second).getByText('Calidad de datos')).toBeInTheDocument();
        expect(within(third).getByText('Falso positivo')).toBeInTheDocument();
        expect(within(first).getByRole('link', {name: 'Investigar anomalía de M-109'})).toHaveAttribute('href', '/anomalies/a-109');
    });

    it('shows how many anomalies there are of each option in the filters', async () =>
    {
        mockApi(routes());

        await renderApp('/anomalies');

        const typeFilter = await screen.findByRole('combobox', {name: 'Tipo'});
        expect(within(typeFilter).getByRole('option', {name: 'Anomalía real (1)'})).toBeInTheDocument();
        expect(within(typeFilter).getByRole('option', {name: 'Explicable (0)'})).toBeInTheDocument();
        expect(within(screen.getByRole('combobox', {name: 'Severidad'})).getByRole('option', {name: 'Alta (2)'})).toBeInTheDocument();
    });

    it('filters through the URL and keeps the original rank', async () =>
    {
        mockApi(routes());
        const user = userEvent.setup();

        const router = await renderApp('/anomalies');

        await user.selectOptions(await screen.findByRole('combobox', {name: 'Tipo'}), 'FALSE_POSITIVE');

        expect(router.state.location.search).toBe('?type=FALSE_POSITIVE');
        expect(cards()).toHaveLength(1);
        expect(within(cards()[0]).getByText('M-106')).toBeInTheDocument();
        expect(within(cards()[0]).getByText('#3')).toBeInTheDocument();
        expect(screen.getByText('1 de 3 anomalías')).toBeInTheDocument();

        await user.selectOptions(screen.getByRole('combobox', {name: 'Tipo'}), '');

        expect(router.state.location.search).toBe('');
        expect(cards()).toHaveLength(3);
    });

    it('opens with the filters of the URL and can clear them when nothing matches', async () =>
    {
        mockApi(routes());
        const user = userEvent.setup();

        await renderApp('/anomalies?type=EXPLAINABLE_ANOMALY&severity=HIGH');

        expect(await screen.findByText('Ninguna anomalía coincide con los filtros')).toBeInTheDocument();
        expect(screen.getByRole('combobox', {name: 'Tipo'})).toHaveValue('EXPLAINABLE_ANOMALY');

        await user.click(screen.getAllByRole('button', {name: 'Limpiar filtros'})[0]);

        expect(cards()).toHaveLength(3);
    });

    it('invites to run an analysis when there are no anomalies yet', async () =>
    {
        mockApi(routes({
            'GET /ai/analysis/latest': apiError(404, 'NO_ANALYSIS_YET'),
            'GET /anomalies': ok({analysisId: null, data: []}),
        }));

        await renderApp('/anomalies');

        expect(await screen.findByText('Aún no hay anomalías')).toBeInTheDocument();
        expect(screen.getAllByRole('button', {name: 'Ejecutar análisis IA'}).length).toBeGreaterThan(1);
    });
});
