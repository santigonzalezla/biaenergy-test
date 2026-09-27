import styles from './pageheader.module.css';
import type {ReactNode} from 'react';

interface PageHeaderProps {
    title: string;
    description?: string;
    actions?: ReactNode;
}

const PageHeader = ({title, description, actions}: PageHeaderProps) =>
{
    return (
        <header className={styles.header}>
            <div className={styles.text}>
                <h1 className={styles.title}>{title}</h1>
                {description && <p className={styles.description}>{description}</p>}
            </div>
            {actions && <div className={styles.actions}>{actions}</div>}
        </header>
    );
}

export default PageHeader;
