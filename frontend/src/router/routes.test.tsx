import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {createMemoryRouter, RouterProvider, type InitialEntry} from 'react-router';
import AuthProvider from '@/context/AuthContext.tsx';
import ThemeProvider from '@/context/ThemeContext.tsx';
import {routes} from '@/router/routes.tsx';
import {clearSession, readSession, saveSession} from '@/lib/session.ts';

const signIn = () => saveSession({
    token: 'valid.jwt.token',
    expiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
    user: {id: 'f90768ae-b4ed-425e-bdbd-918714c97484', email: 'admin@bia.app', name: 'Admin BIA'},
});

const renderAt = (entry: InitialEntry) =>
{
    const router = createMemoryRouter(routes, {initialEntries: [entry]});

    render(
        <ThemeProvider>
            <AuthProvider>
                <RouterProvider router={router}/>
            </AuthProvider>
        </ThemeProvider>
    );

    return router;
}

describe('routes', () =>
{
    beforeEach(() =>
    {
        clearSession();
        localStorage.clear();
    });

    it('sends anonymous visitors to the login and remembers where they were going', () =>
    {
        const router = renderAt('/anomalies?type=REAL_ANOMALY');

        expect(router.state.location.pathname).toBe('/login');
        expect(router.state.location.state.from.pathname).toBe('/anomalies');
        expect(screen.getByRole('heading', {name: 'Ingresar'})).toBeInTheDocument();
    });

    it('opens the dashboard from the root inside the layout', () =>
    {
        signIn();

        const router = renderAt('/');

        expect(router.state.location.pathname).toBe('/dashboard');
        expect(screen.getByRole('heading', {name: 'Dashboard'})).toBeInTheDocument();
        expect(screen.getByRole('link', {name: 'Dashboard'})).toHaveAttribute('aria-current', 'page');
        expect(document.title).toBe('Dashboard · Bia Energy · Anomaly Center');
    });

    it('takes an authenticated user away from the login back to the original page', () =>
    {
        signIn();

        const router = renderAt({pathname: '/login', state: {from: {pathname: '/anomalies', search: '?type=REAL_ANOMALY'}}});

        expect(router.state.location.pathname).toBe('/anomalies');
        expect(router.state.location.search).toBe('?type=REAL_ANOMALY');
    });

    it('shows the route title in the breadcrumb for nested pages', () =>
    {
        signIn();

        renderAt('/meters/5c0b1c8e-8f3f-4d7a-9a57-0e6d0f6b8a10');

        expect(screen.getByRole('navigation', {name: 'Ruta actual'})).toHaveTextContent('Detalle del medidor');
        expect(screen.getByRole('link', {name: 'Medidores'})).toHaveAttribute('aria-current', 'page');
    });

    it('renders the 404 page for unknown routes', () =>
    {
        signIn();

        renderAt('/does-not-exist');

        expect(screen.getByRole('heading', {name: 'Esta página no existe'})).toBeInTheDocument();
    });

    it('logs out from the user menu and returns to the login', async () =>
    {
        signIn();
        const user = userEvent.setup();

        const router = renderAt('/dashboard');

        await user.click(screen.getByRole('button', {name: /Admin BIA/}));
        await user.click(screen.getByRole('menuitem', {name: 'Cerrar sesión'}));

        expect(router.state.location.pathname).toBe('/login');
        expect(readSession()).toBeNull();
    });

    it('toggles the theme from the top bar', async () =>
    {
        signIn();
        const user = userEvent.setup();

        renderAt('/dashboard');

        await user.click(screen.getByRole('button', {name: 'Cambiar a modo oscuro'}));

        expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
        expect(screen.getByRole('button', {name: 'Cambiar a modo claro'})).toBeInTheDocument();
    });
});
