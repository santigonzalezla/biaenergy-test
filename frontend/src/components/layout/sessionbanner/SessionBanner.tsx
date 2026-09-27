import styles from './sessionbanner.module.css';
import {Clock} from 'lucide-react';
import {useAuth} from '@/context/auth.ts';

const SessionBanner = () =>
{
    const {isExpiringSoon, logout} = useAuth();

    if (!isExpiringSoon) return null;

    return (
        <div className={styles.banner} role='status'>
            <Clock size={16} className={styles.icon}/>
            <p className={styles.message}>
                Tu sesión expira en menos de 5 minutos. Guarda lo que estés haciendo y vuelve a ingresar.
            </p>
            <button type='button' className={styles.action} onClick={logout}>
                Ingresar de nuevo
            </button>
        </div>
    );
}

export default SessionBanner;
