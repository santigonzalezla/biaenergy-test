import {Navigate, type RouteObject} from 'react-router';
import AppLayout from '@/components/layout/applayout/AppLayout.tsx';
import ProtectedRoute from '@/guards/ProtectedRoute.tsx';
import PublicOnlyRoute from '@/guards/PublicOnlyRoute.tsx';
import LoginPage from '@/pages/login/LoginPage.tsx';

const page = (load: () => Promise<{default: RouteObject['Component']}>) => async () => ({Component: (await load()).default});

export const routes: RouteObject[] = [
    {
        element: <PublicOnlyRoute/>,
        children: [
            {path: '/login', element: <LoginPage/>, handle: {title: 'Ingresar'}},
        ],
    },
    {
        element: <ProtectedRoute/>,
        children: [
            {
                element: <AppLayout/>,
                children: [
                    {index: true, element: <Navigate to='/dashboard' replace/>},
                    {path: '/dashboard', lazy: page(() => import('@/pages/dashboard/DashboardPage.tsx')), handle: {title: 'Dashboard'}},
                    {path: '/meters', lazy: page(() => import('@/pages/meters/MetersPage.tsx')), handle: {title: 'Medidores'}},
                    {path: '/meters/:meterId', lazy: page(() => import('@/pages/meterdetail/MeterDetailPage.tsx')), handle: {title: 'Detalle del medidor'}},
                    {path: '/anomalies', lazy: page(() => import('@/pages/anomalies/AnomaliesPage.tsx')), handle: {title: 'Anomalías IA'}},
                    {path: '/anomalies/:anomalyId', lazy: page(() => import('@/pages/anomalydetail/AnomalyDetailPage.tsx')), handle: {title: 'Investigación'}},
                    {path: '/import', lazy: page(() => import('@/pages/import/ImportPage.tsx')), handle: {title: 'Importar datos'}},
                    {path: '*', lazy: page(() => import('@/pages/notfound/NotFoundPage.tsx')), handle: {title: 'No encontrada'}},
                ],
            },
        ],
    },
];
