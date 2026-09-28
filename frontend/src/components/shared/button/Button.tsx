import styles from './button.module.css';
import type {ButtonHTMLAttributes, ReactNode} from 'react';
import {Loader2} from 'lucide-react';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
    size?: 'sm' | 'md';
    isLoading?: boolean;
    icon?: ReactNode;
}

const Button = ({variant = 'secondary', size = 'md', isLoading = false, icon, children, className = '', disabled, type = 'button', ...rest}: ButtonProps) =>
{
    return (
        <button
            type={type}
            className={`${styles.button} ${styles[variant]} ${styles[size]} ${className}`}
            disabled={disabled || isLoading}
            aria-busy={isLoading}
            {...rest}
        >
            {isLoading ? <Loader2 size={16} className={styles.spinner}/> : icon}
            {children}
        </button>
    );
}

export default Button;
