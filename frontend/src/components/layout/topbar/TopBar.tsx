import styles from './topbar.module.css';
import {ChevronRight, Menu, Moon, Sun} from 'lucide-react';
import RunAnalysisButton from '@/components/analysis/runanalysisbutton/RunAnalysisButton.tsx';
import UserMenu from '@/components/layout/usermenu/UserMenu.tsx';
import {useTheme} from '@/context/theme.ts';

interface TopBarProps {
    title: string;
    onOpenMenu: () => void;
}

const TopBar = ({title, onOpenMenu}: TopBarProps) =>
{
    const {theme, toggleTheme} = useTheme();
    const isDark = theme === 'dark';

    return (
        <header className={styles.topbar}>
            <div className={styles.left}>
                <button type='button' className={`${styles.iconButton} ${styles.menuButton}`} onClick={onOpenMenu} aria-label='Abrir menú'>
                    <Menu size={18}/>
                </button>
                <nav className={styles.breadcrumb} aria-label='Ruta actual'>
                    <span className={styles.root}>Bia Energy</span>
                    <ChevronRight size={14} className={styles.separator}/>
                    <span className={styles.current}>{title}</span>
                </nav>
            </div>

            <div className={styles.right}>
                <RunAnalysisButton size='sm'/>
                <button
                    type='button'
                    className={styles.iconButton}
                    onClick={toggleTheme}
                    aria-label={isDark ? 'Cambiar a modo claro' : 'Cambiar a modo oscuro'}
                    title={isDark ? 'Modo claro' : 'Modo oscuro'}
                >
                    {isDark ? <Sun size={18}/> : <Moon size={18}/>}
                </button>
                <span className={styles.divider}/>
                <UserMenu/>
            </div>
        </header>
    );
}

export default TopBar;
