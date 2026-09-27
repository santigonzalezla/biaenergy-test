import styles from './usermenu.module.css';
import {useEffect, useRef, useState} from 'react';
import {ChevronDown, LogOut} from 'lucide-react';
import {useAuth} from '@/context/auth.ts';

const initialsOf = (name: string) =>
{
    return name.split(/\s+/).filter(Boolean).slice(0, 2).map(part => part[0].toUpperCase()).join('');
}

const UserMenu = () =>
{
    const {user, logout} = useAuth();
    const [isOpen, setIsOpen] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);

    useEffect(() =>
    {
        if (!isOpen) return;

        const closeOnOutsideClick = (event: MouseEvent) =>
        {
            if (!containerRef.current?.contains(event.target as Node)) setIsOpen(false);
        };

        const closeOnEscape = (event: KeyboardEvent) =>
        {
            if (event.key === 'Escape') setIsOpen(false);
        };

        document.addEventListener('mousedown', closeOnOutsideClick);
        document.addEventListener('keydown', closeOnEscape);

        return () =>
        {
            document.removeEventListener('mousedown', closeOnOutsideClick);
            document.removeEventListener('keydown', closeOnEscape);
        };
    }, [isOpen]);

    if (!user) return null;

    return (
        <div className={styles.container} ref={containerRef}>
            <button
                type='button'
                className={styles.trigger}
                onClick={() => setIsOpen(previous => !previous)}
                aria-haspopup='menu'
                aria-expanded={isOpen}
            >
                <span className={styles.avatar}>{initialsOf(user.name)}</span>
                <span className={styles.identity}>
                    <span className={styles.name}>{user.name}</span>
                    <span className={styles.email}>{user.email}</span>
                </span>
                <ChevronDown size={16} className={styles.chevron}/>
            </button>

            {isOpen && (
                <div className={styles.menu} role='menu'>
                    <div className={styles.menuHeader}>
                        <span className={styles.name}>{user.name}</span>
                        <span className={styles.email}>{user.email}</span>
                    </div>
                    <button type='button' role='menuitem' className={styles.menuItem} onClick={logout}>
                        <LogOut size={16}/>
                        Cerrar sesión
                    </button>
                </div>
            )}
        </div>
    );
}

export default UserMenu;
