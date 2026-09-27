import {createBrowserRouter, RouterProvider} from 'react-router';
import {routes} from '@/router/routes.tsx';

const router = createBrowserRouter(routes);

const AppRouter = () =>
{
    return <RouterProvider router={router}/>;
}

export default AppRouter;
