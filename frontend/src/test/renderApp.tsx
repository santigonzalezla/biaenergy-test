import {render} from '@testing-library/react';
import {createMemoryRouter, RouterProvider, type InitialEntry} from 'react-router';
import AuthProvider from '@/context/AuthContext.tsx';
import ThemeProvider from '@/context/ThemeContext.tsx';
import {routes} from '@/router/routes.tsx';

export const renderApp = (entry: InitialEntry) =>
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
