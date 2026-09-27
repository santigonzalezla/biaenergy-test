import {Navigate, Outlet, useLocation, type Location} from 'react-router';
import {useAuth} from '@/context/auth.ts';

interface RedirectState {
    from?: Location;
}

const PublicOnlyRoute = () =>
{
    const {isAuthenticated} = useAuth();
    const location = useLocation();

    if (!isAuthenticated) return <Outlet/>;

    const from = (location.state as RedirectState | null)?.from;
    const target = from ? `${from.pathname}${from.search}` : '/dashboard';

    return <Navigate to={target} replace/>;
}

export default PublicOnlyRoute;
