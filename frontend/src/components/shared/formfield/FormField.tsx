import styles from './formfield.module.css';
import type {ReactNode} from 'react';

interface FormFieldProps {
    id: string;
    label: string;
    hint?: string;
    error?: string;
    children: ReactNode;
}

const FormField = ({id, label, hint, error, children}: FormFieldProps) =>
{
    return (
        <div className={`${styles.field} ${error ? styles.invalid : ''}`}>
            <label htmlFor={id} className={styles.label}>{label}</label>
            {children}
            {error
                ? <span className={styles.error} id={`${id}-error`}>{error}</span>
                : hint && <span className={styles.hint}>{hint}</span>}
        </div>
    );
}

export default FormField;
