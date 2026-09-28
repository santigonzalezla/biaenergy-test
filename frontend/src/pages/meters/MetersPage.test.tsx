import {screen, waitFor, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type {Meter, Page} from '@/interfaces/interfaces.ts';
import {clearSession} from '@/lib/session.ts';
import {analysis, meterPage, signIn} from '@/test/fixtures.ts';
import {apiError, mockApi, ok, type MockRoutes} from '@/test/mockApi.ts';
import {renderApp} from '@/test/renderApp.tsx';

const listReply = ({path}: {path: string}): Page<Meter> =>
{
    const query = new URLSearchParams(path.split('?')[1]);
    const status = query.get('status');
    const search = query.get('search');
    const page = meterPage();
    const data = page.data
        .filter(meter => !status || meter.status === status)
        .filter(meter => !search || meter.code.includes(search));

    return {...page, data, total: data.length, limit: 20};
}

const routes = (extra: MockRoutes = {}): MockRoutes => ({
    'GET /ai/analysis/latest': ok(analysis()),
    'GET /meters': ok(listReply),
    ...extra,
});

const meterRequests = (fetchMock: ReturnType<typeof mockApi>) =>
    fetchMock.mock.calls.map(([url]) => new URL(String(url))).filter(url => url.pathname === '/api/meters');

describe('MetersPage', () =>
{
    beforeEach(() =>
    {
        clearSession();
        signIn();
    });

    it('lists the meters with their status and recent consumption', async () =>
    {
        mockApi(routes());

        renderApp('/meters');

        const row = (await screen.findByText('M-109')).closest('tr')!;
        expect(within(row).getByText('Crítico')).toBeInTheDocument();
        expect(within(row).getByText('+109,8 %')).toBeInTheDocument();
        expect(screen.getByText('Mostrando', {exact: false})).toHaveTextContent('Mostrando 1–4 de 4 medidores');
    });

    it('filters by status through the URL', async () =>
    {
        const fetchMock = mockApi(routes());
        const user = userEvent.setup();

        const router = renderApp('/meters');
        await screen.findByText('M-101');

        await user.click(screen.getByRole('button', {name: 'Crítico'}));

        await waitFor(() => expect(screen.queryByText('M-101')).not.toBeInTheDocument());
        expect(screen.getByText('M-109')).toBeInTheDocument();
        expect(router.state.location.search).toContain('status=CRITICAL');
        expect(meterRequests(fetchMock).at(-1)?.searchParams.get('status')).toBe('CRITICAL');
    });

    it('debounces the search before asking the API', async () =>
    {
        const fetchMock = mockApi(routes());
        const user = userEvent.setup();

        renderApp('/meters');
        await screen.findByText('M-101');
        const requestsBefore = meterRequests(fetchMock).length;

        await user.type(screen.getByRole('searchbox', {name: 'Buscar medidores'}), 'M-112');

        await waitFor(() => expect(screen.queryByText('M-101')).not.toBeInTheDocument());
        expect(meterRequests(fetchMock).length).toBe(requestsBefore + 1);
        expect(meterRequests(fetchMock).at(-1)?.searchParams.get('search')).toBe('M-112');
    });

    it('toggles the sort direction when clicking the same column', async () =>
    {
        const fetchMock = mockApi(routes());
        const user = userEvent.setup();

        renderApp('/meters');
        await screen.findByText('M-101');

        await user.click(screen.getByRole('button', {name: 'Código'}));

        await waitFor(() => expect(meterRequests(fetchMock).at(-1)?.searchParams.get('sortDir')).toBe('desc'));
        expect(screen.getByRole('columnheader', {name: 'Código'})).toHaveAttribute('aria-sort', 'descending');
    });

    it('validates and creates a meter', async () =>
    {
        const created = {...meterPage().data[0], id: 'm-113', code: 'M-113', name: 'Compresores'};
        const fetchMock = mockApi(routes({'POST /meters': ok(created, 201)}));
        const user = userEvent.setup();

        renderApp('/meters');
        await screen.findByText('M-101');

        await user.click(screen.getByRole('button', {name: 'Nuevo medidor'}));
        const dialog = screen.getByRole('dialog', {name: 'Nuevo medidor'});

        await user.type(within(dialog).getByLabelText('Código'), 'm 113');
        await user.click(within(dialog).getByRole('button', {name: 'Crear medidor'}));

        expect(within(dialog).getByText('Usa de 2 a 20 caracteres: letras, números o guion.')).toBeInTheDocument();
        expect(within(dialog).getByText('El nombre es obligatorio (máximo 120 caracteres).')).toBeInTheDocument();

        await user.clear(within(dialog).getByLabelText('Código'));
        await user.type(within(dialog).getByLabelText('Código'), 'm-113');
        await user.type(within(dialog).getByLabelText('Nombre'), 'Compresores');
        await user.type(within(dialog).getByLabelText('Potencia contratada (kW)'), '350');
        await user.click(within(dialog).getByRole('button', {name: 'Crear medidor'}));

        await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());

        const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!;
        expect(JSON.parse(post[1]!.body as string)).toEqual({
            code: 'M-113',
            name: 'Compresores',
            location: '',
            sector: '',
            nominalVoltage: 220,
            contractedPowerKw: 350,
        });
    });

    it('shows the duplicate code error on the code field', async () =>
    {
        mockApi(routes({'POST /meters': apiError(409, 'METER_DUPLICATE_CODE')}));
        const user = userEvent.setup();

        renderApp('/meters');
        await screen.findByText('M-101');

        await user.click(screen.getByRole('button', {name: 'Nuevo medidor'}));
        const dialog = screen.getByRole('dialog', {name: 'Nuevo medidor'});
        await user.type(within(dialog).getByLabelText('Código'), 'M-109');
        await user.type(within(dialog).getByLabelText('Nombre'), 'Duplicado');
        await user.click(within(dialog).getByRole('button', {name: 'Crear medidor'}));

        expect(await within(dialog).findByText('Ya existe un medidor con ese código.')).toBeInTheDocument();
    });

    it('edits a meter without allowing to change its code', async () =>
    {
        const fetchMock = mockApi(routes({'PATCH /meters/m-109': ok({...meterPage().data[2], name: 'Compresores norte'})}));
        const user = userEvent.setup();

        renderApp('/meters');
        await user.click(await screen.findByRole('button', {name: 'Editar M-109'}));

        const dialog = screen.getByRole('dialog', {name: 'Editar medidor · M-109'});
        expect(within(dialog).getByLabelText('Código')).toBeDisabled();

        await user.clear(within(dialog).getByLabelText('Nombre'));
        await user.type(within(dialog).getByLabelText('Nombre'), 'Compresores norte');
        await user.click(within(dialog).getByRole('button', {name: 'Guardar cambios'}));

        await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
        const patch = fetchMock.mock.calls.find(([, init]) => init?.method === 'PATCH')!;
        expect(JSON.parse(patch[1]!.body as string)).toMatchObject({name: 'Compresores norte'});
        expect(JSON.parse(patch[1]!.body as string)).not.toHaveProperty('code');
    });

    it('asks for confirmation before deleting and refreshes the list', async () =>
    {
        const fetchMock = mockApi(routes({'DELETE /meters/m-104': {status: 204, reply: null}}));
        const user = userEvent.setup();

        renderApp('/meters');
        await user.click(await screen.findByRole('button', {name: 'Eliminar M-104'}));

        const dialog = screen.getByRole('dialog', {name: 'Eliminar M-104'});
        const requestsBefore = meterRequests(fetchMock).length;
        await user.click(within(dialog).getByRole('button', {name: 'Eliminar medidor'}));

        await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
        expect(fetchMock.mock.calls.some(([url, init]) => init?.method === 'DELETE' && String(url).endsWith('/meters/m-104'))).toBe(true);
        expect(meterRequests(fetchMock).length).toBe(requestsBefore + 1);
    });

    it('opens the meter detail when clicking a row', async () =>
    {
        mockApi(routes({'GET /meters/m-109': ok(meterPage().data[2])}));
        const user = userEvent.setup();

        const router = renderApp('/meters');
        await user.click(await screen.findByText('Medidor M-109'));

        expect(router.state.location.pathname).toBe('/meters/m-109');
    });
});
