import {screen, waitFor, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {clearSession} from '@/lib/session.ts';
import {analysis, anomalyDetail, readingSeries, signIn} from '@/test/fixtures.ts';
import {apiError, mockApi, ok, type MockRoutes} from '@/test/mockApi.ts';
import {renderApp} from '@/test/renderApp.tsx';

const routes = (extra: MockRoutes = {}): MockRoutes => ({
    'GET /ai/analysis/latest': ok(analysis()),
    'GET /anomalies/a-109': ok(anomalyDetail()),
    'GET /meters/m-109/readings': ok(readingSeries()),
    ...extra,
});

const section = (name: string) => screen.getByRole('region', {name});

describe('AnomalyDetailPage', () =>
{
    beforeEach(() =>
    {
        clearSession();
        signIn();
    });

    it('explains the anomaly with its metrics, evidence and recommended action', async () =>
    {
        mockApi(routes());

        renderApp('/anomalies/a-109');

        expect(await screen.findByRole('heading', {name: /M-109/})).toBeInTheDocument();
        expect(screen.getByText('44,1 → 92,8 kWh/h')).toBeInTheDocument();
        expect(screen.getByText('+110,5 %')).toBeInTheDocument();
        expect(screen.getByText('90 %')).toBeInTheDocument();

        expect(within(section('Explicación IA')).getByText(/Aumento sostenido del consumo del 110,5 %/)).toBeInTheDocument();
        expect(within(section('Acción recomendada')).getByText('Revisar físicamente los equipos conectados a este medidor.')).toBeInTheDocument();

        const signals = within(section('Señales detectadas'));
        expect(signals.getByText('Aumento sostenido de consumo')).toBeInTheDocument();
        expect(signals.getByText('Factor de potencia bajo (< 0,80)')).toBeInTheDocument();
        expect(signals.getByText('0,74')).toBeInTheDocument();

        const variables = within(section('Variables que cambiaron'));
        expect(variables.getByRole('rowheader', {name: 'Consumo'})).toBeInTheDocument();
        expect(variables.getByText('+116,9 %')).toBeInTheDocument();

        expect(screen.getByText('No explica el cambio: no hay un evento operativo que lo justifique.')).toBeInTheDocument();
        expect(screen.getByRole('link', {name: /Ver medidor/})).toHaveAttribute('href', '/meters/m-109');
    });

    it('offers only the transitions the backend allows for the current status', async () =>
    {
        mockApi(routes({'GET /anomalies/a-109': ok(anomalyDetail({status: 'RESOLVED', resolutionNote: 'Motor desconectado.'}))}));

        renderApp('/anomalies/a-109');

        expect(await screen.findByRole('button', {name: 'Reabrir'})).toBeInTheDocument();
        expect(screen.queryByRole('button', {name: 'Marcar como resuelta'})).not.toBeInTheDocument();
        expect(screen.getByText('Motor desconectado.')).toBeInTheDocument();
    });

    it('moves the anomaly to investigation with a note and shows the new status', async () =>
    {
        const fetchMock = mockApi(routes({
            'PATCH /anomalies/a-109/status': ok(({body}) => anomalyDetail({status: 'INVESTIGATING', resolutionNote: (body as {note: string}).note})),
        }));
        const user = userEvent.setup();

        renderApp('/anomalies/a-109');

        await user.type(await screen.findByRole('textbox', {name: /Nota/}), '  Técnico asignado a la nave C  ');
        await user.click(screen.getByRole('button', {name: 'Marcar en investigación'}));

        await waitFor(() => expect(screen.getByRole('button', {name: 'Reabrir'})).toBeInTheDocument());
        expect(within(screen.getByText('Estado actual').parentElement!).getByText('En investigación')).toBeInTheDocument();
        expect(screen.getByText('Técnico asignado a la nave C')).toBeInTheDocument();
        expect(screen.getByRole('textbox', {name: /Nota/})).toHaveValue('');

        const patch = fetchMock.mock.calls.find(([, init]) => init?.method === 'PATCH')!;
        expect(JSON.parse(patch[1]!.body as string)).toEqual({status: 'INVESTIGATING', note: 'Técnico asignado a la nave C'});
    });

    it('blocks notes longer than 500 characters', async () =>
    {
        mockApi(routes());
        const user = userEvent.setup();

        renderApp('/anomalies/a-109');

        const note = await screen.findByRole('textbox', {name: /Nota/});
        await user.click(note);
        await user.paste('x'.repeat(501));

        expect(screen.getByText('501/500')).toBeInTheDocument();
        expect(screen.getByRole('button', {name: 'Descartar'})).toBeDisabled();
    });

    it('keeps the status when the backend rejects the transition', async () =>
    {
        mockApi(routes({'PATCH /anomalies/a-109/status': apiError(409, 'INVALID_STATUS_TRANSITION')}));
        const user = userEvent.setup();

        renderApp('/anomalies/a-109');

        await user.click(await screen.findByRole('button', {name: 'Descartar'}));

        await waitFor(() => expect(screen.getByRole('button', {name: 'Descartar'})).toBeEnabled());
        expect(within(screen.getByText('Estado actual').parentElement!).getByText('Abierta')).toBeInTheDocument();
    });

    it('shows a not found state for an unknown anomaly', async () =>
    {
        mockApi(routes({'GET /anomalies/nope': apiError(404, 'ANOMALY_NOT_FOUND')}));

        renderApp('/anomalies/nope');

        expect(await screen.findByText('Anomalía no encontrada')).toBeInTheDocument();
        expect(screen.getByRole('link', {name: /Volver a anomalías/})).toHaveAttribute('href', '/anomalies');
    });
});
