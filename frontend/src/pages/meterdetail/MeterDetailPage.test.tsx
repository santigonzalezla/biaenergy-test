import {screen, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type {HourlyProfile} from '@/interfaces/interfaces.ts';
import {clearSession} from '@/lib/session.ts';
import {analysis, anomalyDetail, anomalyList, CHANGE_AT, meterPage, readingSeries, signIn} from '@/test/fixtures.ts';
import {apiError, mockApi, ok, type MockRoutes} from '@/test/mockApi.ts';
import {renderApp} from '@/test/renderApp.tsx';

const m109 = () => meterPage().data[2];

const profile = ({path}: {path: string}): HourlyProfile =>
{
    const query = new URLSearchParams(path.split('?')[1]);
    const after = query.get('from') === CHANGE_AT;

    return {
        from: query.get('from')!,
        to: query.get('to')!,
        timezone: 'America/Bogota',
        hours: Array.from({length: 24}, (_, hour) => ({hour, avgKwh: after ? 92 : 44, samples: 12})),
    };
}

const routes = (extra: MockRoutes = {}): MockRoutes => ({
    'GET /ai/analysis/latest': ok(analysis()),
    'GET /meters/m-109': ok(m109()),
    'GET /anomalies': ok(anomalyList()),
    'GET /anomalies/a-109': ok(anomalyDetail()),
    'GET /meters/m-109/readings': ok(readingSeries()),
    'GET /meters/m-109/events': ok({from: '', to: '', data: [{id: 'e-1', type: 'UNKNOWN', timestamp: CHANGE_AT, description: 'No operational event reported'}]}),
    'GET /meters/m-109/profile': ok(profile),
    ...extra,
});

const requestedPaths = (fetchMock: ReturnType<typeof mockApi>) => fetchMock.mock.calls.map(([url]) => new URL(String(url)));

describe('MeterDetailPage', () =>
{
    beforeEach(() =>
    {
        clearSession();
        signIn();
    });

    it('shows the meter, its active anomaly and the stats around the change point', async () =>
    {
        mockApi(routes());

        renderApp('/meters/m-109');

        expect(await screen.findByRole('heading', {name: 'Medidor M-109'})).toBeInTheDocument();
        expect(screen.getByText('Crítico')).toBeInTheDocument();

        const callout = await screen.findByRole('region', {name: 'Anomalía detectada en el último análisis'});
        expect(within(callout).getByText('Anomalía real')).toBeInTheDocument();
        expect(within(callout).getByText('+110,5 % de consumo')).toBeInTheDocument();
        expect(within(callout).getByText(/12 de sept de 2026, 14:00/)).toBeInTheDocument();
        expect(within(callout).getByRole('link', {name: /Ver investigación/})).toHaveAttribute('href', '/anomalies/a-109');

        expect(await screen.findByText('44,1 kWh/h')).toBeInTheDocument();
        expect(screen.getByText('92,8 kWh/h')).toBeInTheDocument();
    });

    it('compares each electrical variable before and after the change with its threshold', async () =>
    {
        mockApi(routes());
        const user = userEvent.setup();

        renderApp('/meters/m-109');

        const panel = await screen.findByRole('tabpanel', {name: 'Factor de potencia'});
        expect(within(panel).getByText('0,94')).toBeInTheDocument();
        expect(within(panel).getAllByText('0,74')).not.toHaveLength(0);
        expect(within(panel).getByText('Mínimo aceptable: 0,80')).toBeInTheDocument();

        await user.click(screen.getByRole('tab', {name: 'Corriente'}));
        const current = screen.getByRole('tabpanel', {name: 'Corriente'});
        expect(within(current).getByText('+100,4 %')).toBeInTheDocument();
        expect(within(current).getByText('Sobrecorriente: > 1,5 × pico previo (363 A)')).toBeInTheDocument();

        await user.click(screen.getByRole('tab', {name: 'Voltaje'}));
        expect(screen.getByText('Banda ±5 %: 209–231 V')).toBeInTheDocument();
    });

    it('marks the event that coincides with the change but does not explain it', async () =>
    {
        mockApi(routes());

        renderApp('/meters/m-109');

        expect(await screen.findByText('Coincide con el cambio, pero no lo explica')).toBeInTheDocument();
        expect(screen.getByText('No operational event reported')).toBeInTheDocument();
    });

    it('asks the readings for the selected range ending at the last reading', async () =>
    {
        const fetchMock = mockApi(routes());
        const user = userEvent.setup();

        renderApp('/meters/m-109');
        await screen.findByRole('tabpanel', {name: 'Factor de potencia'});

        await user.click(screen.getByRole('button', {name: '3 días'}));

        const readingRequests = requestedPaths(fetchMock).filter(url => url.pathname.endsWith('/readings'));
        const last = readingRequests.at(-1)!;
        expect(last.searchParams.get('to')).toBe('2026-09-15T05:00:00.000Z');
        expect(last.searchParams.get('from')).toBe('2026-09-12T05:00:00.000Z');

        const profileRequests = requestedPaths(fetchMock).filter(url => url.pathname.endsWith('/profile'));
        expect(profileRequests.map(url => url.searchParams.get('to'))).toContain(CHANGE_AT);
        expect(profileRequests.map(url => url.searchParams.get('from'))).toContain(CHANGE_AT);
    });

    it('works for a meter without anomalies in the latest analysis', async () =>
    {
        const m101 = meterPage().data[0];
        mockApi({
            'GET /ai/analysis/latest': ok(analysis()),
            'GET /meters/m-101': ok(m101),
            'GET /anomalies': ok(anomalyList()),
            'GET /meters/m-101/readings': ok(readingSeries()),
            'GET /meters/m-101/events': ok({from: '', to: '', data: []}),
            'GET /meters/m-101/profile': ok(profile),
        });

        renderApp('/meters/m-101');

        expect(await screen.findByText('El último análisis no encontró anomalías en este medidor.')).toBeInTheDocument();
        expect(screen.queryByRole('region', {name: 'Anomalía detectada en el último análisis'})).not.toBeInTheDocument();
        expect(await screen.findByText('Sin eventos en el periodo')).toBeInTheDocument();
    });

    it('shows a not found state with a way back', async () =>
    {
        mockApi({
            'GET /ai/analysis/latest': ok(analysis()),
            'GET /meters/unknown': apiError(404, 'METER_NOT_FOUND'),
            'GET /anomalies': ok(anomalyList()),
        });

        renderApp('/meters/unknown');

        expect(await screen.findByText('Medidor no encontrado')).toBeInTheDocument();
        expect(screen.getByRole('link', {name: /Volver a medidores/})).toHaveAttribute('href', '/meters');
    });
});
