import {fireEvent, screen, waitFor, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type {ImportBatch, ImportResult} from '@/interfaces/interfaces.ts';
import {clearSession} from '@/lib/session.ts';
import {analysis, signIn} from '@/test/fixtures.ts';
import {apiError, mockApi, ok, type MockRoutes} from '@/test/mockApi.ts';
import {renderApp} from '@/test/renderApp.tsx';

const csv = (name = 'readings.csv', content = 'meter_id,timestamp\nM-101,2026-09-01 00:00:00') => new File([content], name, {type: 'text/csv'});

const batch = (overrides: Partial<ImportBatch> = {}): ImportBatch => ({
    id: 'b-1', numId: 1, kind: 'READINGS', status: 'COMPLETED', fileName: 'readings.csv', fileSize: 180_000,
    checksum: '750d42f11c2c9a1b', rows: 4032, inserted: 4032, skipped: 0, metersCreated: 12, metersRestored: 0,
    errorCode: null, error: null, userName: 'Admin BIA', createdAt: '2026-09-27T22:00:00Z',
    ...overrides,
});

const importResult = (overrides: Partial<ImportResult> = {}): ImportResult => ({
    batchId: 'b-2', rows: 4032, inserted: 336, skipped: 3696, metersCreated: 1, metersRestored: 0, previouslyImportedAt: null,
    ...overrides,
});

const routes = (extra: MockRoutes = {}): MockRoutes => ({
    'GET /ai/analysis/latest': ok(analysis()),
    'GET /imports?limit=20': ok({data: [batch()]}),
    ...extra,
});

const historyRequests = (fetchMock: ReturnType<typeof mockApi>) =>
    fetchMock.mock.calls.filter(([url]) => String(url).endsWith('/api/imports?limit=20')).length;

const readingsCard = () => screen.getByRole('heading', {name: 'Lecturas'}).closest('section')!;

describe('ImportPage', () =>
{
    beforeEach(() =>
    {
        clearSession();
        signIn();
    });

    it('uploads the readings CSV as multipart and shows the result', async () =>
    {
        const fetchMock = mockApi(routes({'POST /readings/import': ok(importResult())}));
        const user = userEvent.setup();

        await renderApp('/import');

        const card = within(readingsCard());
        await user.upload(card.getByLabelText('Archivo CSV de lecturas'), csv());
        expect(card.getByText('readings.csv')).toBeInTheDocument();

        await user.click(card.getByRole('button', {name: 'Importar archivo'}));

        const result = await card.findByRole('status');
        expect(result).toHaveTextContent('Importación completada');
        expect(within(result).getByText('4.032')).toBeInTheDocument();
        expect(within(result).getByText('3.696')).toBeInTheDocument();

        const [, init] = fetchMock.mock.calls.find(([, request]) => request?.method === 'POST')!;
        expect(init!.body).toBeInstanceOf(FormData);
        expect(((init!.body as FormData).get('file') as File).name).toBe('readings.csv');

        expect(screen.getByText(/Hay datos nuevos/)).toBeInTheDocument();
    });

    it('uploads the events CSV to its own endpoint', async () =>
    {
        const fetchMock = mockApi(routes({'POST /events/import': ok(importResult({rows: 4, inserted: 4, skipped: 0, metersCreated: 0}))}));
        const user = userEvent.setup();

        await renderApp('/import');

        const card = within(screen.getByRole('heading', {name: 'Eventos'}).closest('section')!);
        await user.upload(card.getByLabelText('Archivo CSV de eventos'), csv('events.csv', 'meter_id,event_timestamp,event_type,description'));
        await user.click(card.getByRole('button', {name: 'Importar archivo'}));

        expect(await card.findByRole('status')).toHaveTextContent('Importación completada');
        expect(fetchMock.mock.calls.some(([url, init]) => init?.method === 'POST' && String(url).endsWith('/api/events/import'))).toBe(true);
        expect(fetchMock.mock.calls.some(([url]) => String(url).endsWith('/api/readings/import'))).toBe(false);
    });

    it('lists the invalid rows with their column when the backend rejects the file', async () =>
    {
        mockApi(routes({
            'POST /readings/import': {
                status: 400,
                reply: {error: {code: 'VALIDATION_ERROR', message: 'invalid', details: [
                    {line: 3, column: 'power_factor', message: 'must be between 0 and 1'},
                    {line: 7, column: 'timestamp', message: 'invalid date'},
                ]}},
            },
        }));
        const user = userEvent.setup();

        await renderApp('/import');

        const card = within(readingsCard());
        await user.upload(card.getByLabelText('Archivo CSV de lecturas'), csv());
        await user.click(card.getByRole('button', {name: 'Importar archivo'}));

        const alert = await card.findByRole('alert');
        expect(alert).toHaveTextContent('No se importó nada: hay filas con errores');
        expect(within(alert).getByText('power_factor')).toBeInTheDocument();
        expect(within(alert).getByText('invalid date')).toBeInTheDocument();
        expect(screen.queryByText(/Hay datos nuevos/)).not.toBeInTheDocument();
    });

    it('explains a file with the wrong columns', async () =>
    {
        mockApi(routes({'POST /events/import': apiError(400, 'INVALID_CSV', 'missing required columns: event_type')}));
        const user = userEvent.setup();

        await renderApp('/import');

        const card = within(screen.getByRole('heading', {name: 'Eventos'}).closest('section')!);
        await user.upload(card.getByLabelText('Archivo CSV de eventos'), csv('events.csv'));
        await user.click(card.getByRole('button', {name: 'Importar archivo'}));

        expect(await card.findByRole('alert')).toHaveTextContent('El archivo no tiene el formato esperado: missing required columns: event_type');
    });

    it('rejects files bigger than 10 MB before uploading them', async () =>
    {
        const fetchMock = mockApi(routes());
        const user = userEvent.setup();

        await renderApp('/import');

        const card = within(readingsCard());
        await user.upload(card.getByLabelText('Archivo CSV de lecturas'), csv('huge.csv', 'x'.repeat(10 * 1024 * 1024 + 1)));

        expect(card.getByRole('alert')).toHaveTextContent('El archivo supera el máximo de 10 MB.');
        expect(card.getByRole('button', {name: 'Importar archivo'})).toBeDisabled();
        expect(fetchMock.mock.calls.some(([, request]) => request?.method === 'POST')).toBe(false);
    });

    it('accepts a CSV dropped on the zone and ignores other files', async () =>
    {
        mockApi(routes());

        await renderApp('/import');

        const card = within(readingsCard());
        const zone = card.getByText(/Arrastra el CSV aquí/).closest('label')!;

        fireEvent.drop(zone, {dataTransfer: {files: [new File(['a'], 'photo.png', {type: 'image/png'})]}});
        expect(card.getByRole('alert')).toHaveTextContent('Selecciona un archivo con extensión .csv.');

        fireEvent.drop(zone, {dataTransfer: {files: [csv('dropped.csv')]}});
        await waitFor(() => expect(card.getByText('dropped.csv')).toBeInTheDocument());
        expect(card.getByRole('button', {name: 'Importar archivo'})).toBeEnabled();
    });

    it('shows the history of imports, including the rejected ones', async () =>
    {
        mockApi(routes({'GET /imports?limit=20': ok({data: [
            batch({id: 'b-2', numId: 2, status: 'FAILED', fileName: 'bad.csv', errorCode: 'VALIDATION_ERROR', error: '1 invalid rows; first at line 2, column power_factor'}),
            batch({userName: null}),
        ]})}));

        await renderApp('/import');

        const history = within(screen.getByRole('heading', {name: 'Historial de importaciones'}).closest('section')!);
        const rows = await history.findAllByRole('row');
        expect(rows[1]).toHaveTextContent('bad.csv');
        expect(rows[1]).toHaveTextContent('Rechazado');
        expect(rows[1]).toHaveTextContent('column power_factor');
        expect(rows[2]).toHaveTextContent('4.032 nuevas · 0 omitidas · 12 medidores creados');
        expect(rows[2]).toHaveTextContent('Carga inicial');
    });

    it('refreshes the history after an upload, even when it is rejected', async () =>
    {
        const fetchMock = mockApi(routes({'POST /readings/import': apiError(400, 'EMPTY_CSV')}));
        const user = userEvent.setup();

        await renderApp('/import');
        await waitFor(() => expect(historyRequests(fetchMock)).toBe(1));

        const card = within(readingsCard());
        await user.upload(card.getByLabelText('Archivo CSV de lecturas'), csv());
        await user.click(card.getByRole('button', {name: 'Importar archivo'}));

        expect(await card.findByRole('alert')).toHaveTextContent('El archivo no tiene filas de datos.');
        await waitFor(() => expect(historyRequests(fetchMock)).toBe(2));
    });

    it('tells when the same file was already imported and which meters were restored', async () =>
    {
        mockApi(routes({'POST /readings/import': ok(importResult({inserted: 24, skipped: 4008, metersRestored: 1, previouslyImportedAt: '2026-09-27T22:00:00Z'}))}));
        const user = userEvent.setup();

        await renderApp('/import');

        const card = within(readingsCard());
        await user.upload(card.getByLabelText('Archivo CSV de lecturas'), csv());
        await user.click(card.getByRole('button', {name: 'Importar archivo'}));

        const status = await card.findByRole('status');
        expect(within(status).getByText('Medidores restaurados').nextSibling).toHaveTextContent('1');
        expect(status).toHaveTextContent('Este mismo archivo ya se había importado el 27 de sept de 2026, 17:00');
    });
});
