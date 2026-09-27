import styles from './applayout.module.css';
import {useEffect, useState} from 'react';
import {Outlet, useMatches} from 'react-router';
import SessionBanner from '@/components/layout/sessionbanner/SessionBanner.tsx';
import Sidebar from '@/components/layout/sidebar/Sidebar.tsx';
import TopBar from '@/components/layout/topbar/TopBar.tsx';
import {APP_NAME} from '@/lib/constants.ts';

const COLLAPSED_STORAGE_KEY = 'biaenergy.sidebarCollapsed';

interface RouteHandle {
    title?: string;
}

const AppLayout = () =>
{
    const matches = useMatches();
    const [isCollapsed, setIsCollapsed] = useState(() => localStorage.getItem(COLLAPSED_STORAGE_KEY) === 'true');
    const [isMobileOpen, setIsMobileOpen] = useState(false);

    const title = [...matches].reverse()
        .map(match => (match.handle as RouteHandle | undefined)?.title)
        .find(Boolean) ?? 'Dashboard';

    useEffect(() =>
    {
        document.title = `${title} · ${APP_NAME}`;
    }, [title]);

    useEffect(() =>
    {
        localStorage.setItem(COLLAPSED_STORAGE_KEY, String(isCollapsed));
    }, [isCollapsed]);

    const toggleCollapse = () => setIsCollapsed(previous => !previous);

    return (
        <div className={`${styles.layout} ${isCollapsed ? styles.collapsed : ''}`}>
            <Sidebar
                isCollapsed={isCollapsed}
                isMobileOpen={isMobileOpen}
                onToggleCollapse={toggleCollapse}
                onMobileClose={() => setIsMobileOpen(false)}
            />

            <div className={styles.main}>
                <TopBar title={title} onOpenMenu={() => setIsMobileOpen(true)}/>
                <SessionBanner/>
                <main className={styles.content}>
                    <Outlet/>
                </main>
            </div>
        </div>
    );
}

export default AppLayout;
