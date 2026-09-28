import {act, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {clearSession, notifySessionExpired, readSession, saveSession} from '@/lib/session.ts';
import {renderApp} from '@/test/renderApp.tsx';

const admin = {id: 'f90768ae-b4ed-425e-bdbd-918714c97484', email: 'admin@bia.app', name: 'Admin BIA'};

const jsonResponse = (status: number, body: unknown) =>
    new Response(JSON.stringify(body), {status, headers: {'Content-Type': 'application/json'}});

const renderLogin = (from?: string) => renderApp(from ? {pathname: '/login', state: {from: {pathname: from, search: ''}}} : '/login');

describe('LoginForm', () =>
{
    const fetchMock = vi.fn();

    beforeEach(() =>
    {
        clearSession();
        fetchMock.mockReset();
        vi.stubGlobal('fetch', fetchMock);
    });

    afterEach(() => vi.unstubAllGlobals());

    it('logs in and goes back to the page the user wanted', async () =>
    {
        const expiresAt = new Date(Date.now() + 12 * 60 * 60 * 1000).toISOString();
        fetchMock.mockResolvedValue(jsonResponse(200, {token: 'new.jwt.token', tokenType: 'Bearer', expiresAt, user: admin}));
        const user = userEvent.setup();

        const router = await renderLogin('/anomalies');

        await user.type(screen.getByLabelText('Correo electrónico'), 'admin@bia.app');
        await user.type(screen.getByLabelText('Contraseña'), 'a-long-admin-password');
        await user.click(screen.getByRole('button', {name: 'Ingresar'}));

        expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({email: 'admin@bia.app', password: 'a-long-admin-password'});
        await waitFor(() => expect(router.state.location.pathname).toBe('/anomalies'));
        expect(readSession()?.token).toBe('new.jwt.token');
    });

    it('shows the Spanish error and stays on the login when the credentials are wrong', async () =>
    {
        fetchMock.mockResolvedValue(jsonResponse(401, {error: {code: 'INVALID_CREDENTIALS', message: 'Email or password is incorrect'}}));
        const user = userEvent.setup();

        const router = await renderLogin();

        await user.type(screen.getByLabelText('Correo electrónico'), 'admin@bia.app');
        await user.type(screen.getByLabelText('Contraseña'), 'wrong-password');
        await user.click(screen.getByRole('button', {name: 'Ingresar'}));

        expect(await screen.findByRole('alert')).toHaveTextContent('Correo o contraseña incorrectos.');
        expect(router.state.location.pathname).toBe('/login');
        expect(screen.getByRole('button', {name: 'Ingresar'})).toBeEnabled();
    });

    it('marks the fields rejected by the backend validation', async () =>
    {
        fetchMock.mockResolvedValue(jsonResponse(400, {
            error: {code: 'VALIDATION_ERROR', message: 'invalid', details: {email: 'Email must be a valid address.'}},
        }));
        const user = userEvent.setup();

        await renderLogin();

        await user.type(screen.getByLabelText('Correo electrónico'), 'admin');
        await user.click(screen.getByRole('button', {name: 'Ingresar'}));

        expect(await screen.findByText('Ingresa un correo válido.')).toBeInTheDocument();
        expect(screen.getByLabelText('Correo electrónico')).toHaveAttribute('aria-invalid', 'true');
    });

    it('shows and hides the password', async () =>
    {
        const user = userEvent.setup();

        await renderLogin();

        const password = screen.getByLabelText('Contraseña');
        expect(password).toHaveAttribute('type', 'password');

        await user.click(screen.getByRole('button', {name: 'Mostrar contraseña'}));
        expect(password).toHaveAttribute('type', 'text');

        await user.click(screen.getByRole('button', {name: 'Ocultar contraseña'}));
        expect(password).toHaveAttribute('type', 'password');
    });

    it('tells the user when the session expired', async () =>
    {
        saveSession({token: 'old.jwt.token', expiresAt: new Date(Date.now() + 60_000).toISOString(), user: admin});
        const router = await renderApp('/dashboard');

        act(() => notifySessionExpired());

        expect(router.state.location.pathname).toBe('/login');
        expect(screen.getByRole('status')).toHaveTextContent('Tu sesión expiró. Vuelve a ingresar para continuar.');
    });
});
