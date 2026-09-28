import styles from './card.module.css';
import type {ReactNode} from 'react';

interface CardProps {
    title?: string;
    subtitle?: string;
    actions?: ReactNode;
    children: ReactNode;
    className?: string;
}

const Card = ({title, subtitle, actions, children, className = ''}: CardProps) =>
{
    const hasHeader = Boolean(title || actions);

    return (
        <section className={`${styles.card} ${className}`}>
            {hasHeader && (
                <header className={styles.header}>
                    <div>
                        {title && <h2 className={styles.title}>{title}</h2>}
                        {subtitle && <p className={styles.subtitle}>{subtitle}</p>}
                    </div>
                    {actions && <div className={styles.actions}>{actions}</div>}
                </header>
            )}
            {children}
        </section>
    );
}

export default Card;
