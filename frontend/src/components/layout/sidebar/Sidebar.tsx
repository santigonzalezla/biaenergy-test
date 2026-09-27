import styles from './sidebar.module.css';
import {NavLink} from 'react-router';
import {Gauge, LayoutDashboard, PanelLeftClose, PanelLeftOpen, Sparkles, Upload, X, Zap, type LucideIcon} from 'lucide-react';

interface NavigationItem {
    label: string;
    path: string;
    icon: LucideIcon;
}

interface SidebarProps {
    isCollapsed: boolean;
    isMobileOpen: boolean;
    onToggleCollapse: () => void;
    onMobileClose: () => void;
}

const NAVIGATION: NavigationItem[] = [
    {label: 'Dashboard', path: '/dashboard', icon: LayoutDashboard},
    {label: 'Medidores', path: '/meters', icon: Gauge},
    {label: 'Anomalías IA', path: '/anomalies', icon: Sparkles},
    {label: 'Importar datos', path: '/import', icon: Upload},
];

const Sidebar = ({isCollapsed, isMobileOpen, onToggleCollapse, onMobileClose}: SidebarProps) =>
{
    const CollapseIcon = isCollapsed ? PanelLeftOpen : PanelLeftClose;

    return (
        <>
            <div
                className={`${styles.backdrop} ${isMobileOpen ? styles.backdropVisible : ''}`}
                onClick={onMobileClose}
                aria-hidden='true'
            />

            <aside
                className={`${styles.sidebar} ${isCollapsed ? styles.collapsed : ''} ${isMobileOpen ? styles.mobileOpen : ''}`}
                aria-label='Navegación principal'
            >
                <div className={styles.brand}>
                    <div className={styles.logo}>
                        <Zap size={18} strokeWidth={2.5}/>
                    </div>
                    <div className={styles.brandText}>
                        <span className={styles.wordmark}>bia <strong>energy</strong></span>
                        <span className={styles.product}>Anomaly Center</span>
                    </div>
                    <button type='button' className={styles.closeButton} onClick={onMobileClose} aria-label='Cerrar menú'>
                        <X size={18}/>
                    </button>
                </div>

                <span className={styles.sectionLabel}>Monitoreo</span>

                <nav className={styles.navigation}>
                    {NAVIGATION.map(({label, path, icon: Icon}) => (
                        <NavLink
                            key={path}
                            to={path}
                            title={isCollapsed ? label : undefined}
                            onClick={onMobileClose}
                            className={({isActive}) => `${styles.link} ${isActive ? styles.active : ''}`}
                        >
                            <Icon size={20} strokeWidth={1.75}/>
                            <span className={styles.linkLabel}>{label}</span>
                        </NavLink>
                    ))}
                </nav>

                <button
                    type='button'
                    className={styles.collapseButton}
                    onClick={onToggleCollapse}
                    aria-label={isCollapsed ? 'Expandir menú' : 'Colapsar menú'}
                >
                    <CollapseIcon size={18} strokeWidth={1.75}/>
                    <span className={styles.linkLabel}>Colapsar</span>
                </button>
            </aside>
        </>
    );
}

export default Sidebar;
