import styles from './kpicard.module.css';
import type {ReactNode} from 'react';
import type {LucideIcon} from 'lucide-react';

interface KpiCardProps {
    label: string;
    value: string;
    unit?: string;
    icon: LucideIcon;
    footer?: ReactNode;
    tone?: 'default' | 'danger';
}

const KpiCard = ({label, value, unit, icon: Icon, footer, tone = 'default'}: KpiCardProps) =>
{
    return (
        <article className={`${styles.card} ${styles[tone]}`}>
            <header className={styles.header}>
                <span className={styles.label}>{label}</span>
                <span className={styles.icon}>
                    <Icon size={18} strokeWidth={1.75}/>
                </span>
            </header>
            <p className={styles.value}>
                <span className='tabular'>{value}</span>
                {unit && <span className={styles.unit}>{unit}</span>}
            </p>
            {footer && <div className={styles.footer}>{footer}</div>}
        </article>
    );
}

export default KpiCard;
