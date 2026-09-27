import {Navigate, type RouteObject} from 'react-router';
import AppLayout from '@/components/layout/applayout/AppLayout.tsx';
import ProtectedRoute from '@/guards/ProtectedRoute.tsx';
import PublicOnlyRoute from '@/guards/PublicOnlyRoute.tsx';
import AnomaliesPage from '@/pages/anomalies/AnomaliesPage.tsx';
import AnomalyDetailPage from '@/pages/anomalydetail/AnomalyDetailPage.tsx';
import DashboardPage from '@/pages/dashboard/DashboardPage.tsx';
import ImportPage from '@/pages/import/ImportPage.tsx';
import LoginPage from '@/pages/login/LoginPage.tsx';
import MeterDetailPage from '@/pages/meterdetail/MeterDetailPage.tsx';
import MetersPage from '@/pages/meters/MetersPage.tsx';
import NotFoundPage from '@/pages/notfound/NotFoundPage.tsx';

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
                    {path: '/dashboard', element: <DashboardPage/>, handle: {title: 'Dashboard'}},
                    {path: '/meters', element: <MetersPage/>, handle: {title: 'Medidores'}},
                    {path: '/meters/:meterId', element: <MeterDetailPage/>, handle: {title: 'Detalle del medidor'}},
                    {path: '/anomalies', element: <AnomaliesPage/>, handle: {title: 'Anomalías IA'}},
                    {path: '/anomalies/:anomalyId', element: <AnomalyDetailPage/>, handle: {title: 'Investigación'}},
                    {path: '/import', element: <ImportPage/>, handle: {title: 'Importar datos'}},
                    {path: '*', element: <NotFoundPage/>, handle: {title: 'No encontrada'}},
                ],
            },
        ],
    },
];
