import {render, waitFor} from '@testing-library/react';
import {createMemoryRouter, RouterProvider, type InitialEntry} from 'react-router';
import AuthProvider from '@/context/AuthContext.tsx';
import ThemeProvider from '@/context/ThemeContext.tsx';
import {routes} from '@/router/routes.tsx';

export const renderApp = async (entry: InitialEntry) =>
{
    const router = createMemoryRouter(routes, {initialEntries: [entry]});

    const {container} = render(
        <ThemeProvider>
            <AuthProvider>
                <RouterProvider router={router}/>
            </AuthProvider>
        </ThemeProvider>
    );

    await waitFor(() =>
    {
        expect(router.state.initialized && router.state.navigation.state === 'idle').toBe(true);
        expect(container.childElementCount).toBeGreaterThan(0);
    }, {timeout: 10_000});

    return router;
}
