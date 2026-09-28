import styles from './statemessage.module.css';
import type {ReactNode} from 'react';
import {AlertTriangle, RotateCw, type LucideIcon} from 'lucide-react';

interface StateMessageProps {
    title: string;
    description?: string;
    icon?: LucideIcon;
    variant?: 'empty' | 'error';
    action?: ReactNode;
    onRetry?: () => void;
}

const StateMessage = ({title, description, icon, variant = 'empty', action, onRetry}: StateMessageProps) =>
{
    const Icon = icon ?? AlertTriangle;

    return (
        <div className={`${styles.container} ${styles[variant]}`} role={variant === 'error' ? 'alert' : undefined}>
            <div className={styles.icon}>
                <Icon size={24} strokeWidth={1.75}/>
            </div>
            <p className={styles.title}>{title}</p>
            {description && <p className={styles.description}>{description}</p>}
            {onRetry && (
                <button type='button' className={styles.retry} onClick={onRetry}>
                    <RotateCw size={14}/>
                    Reintentar
                </button>
            )}
            {action}
        </div>
    );
}

export default StateMessage;
