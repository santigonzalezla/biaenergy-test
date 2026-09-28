import styles from './modal.module.css';
import {useEffect, useId, useRef, type ReactNode} from 'react';
import {createPortal} from 'react-dom';
import {X} from 'lucide-react';

interface ModalProps {
    isOpen: boolean;
    title: string;
    description?: string;
    icon?: ReactNode;
    size?: 'sm' | 'md';
    onClose: () => void;
    children: ReactNode;
    footer?: ReactNode;
}

const Modal = ({isOpen, title, description, icon, size = 'md', onClose, children, footer}: ModalProps) =>
{
    const titleId = useId();
    const dialogRef = useRef<HTMLDivElement>(null);
    const onCloseRef = useRef(onClose);

    useEffect(() =>
    {
        onCloseRef.current = onClose;
    }, [onClose]);

    useEffect(() =>
    {
        if (!isOpen) return;

        const previouslyFocused = document.activeElement as HTMLElement | null;
        const firstField = dialogRef.current?.querySelector<HTMLElement>('input:not(:disabled), select, textarea, button:not([data-close])');
        firstField?.focus();

        const closeOnEscape = (event: KeyboardEvent) =>
        {
            if (event.key === 'Escape') onCloseRef.current();
        };

        document.addEventListener('keydown', closeOnEscape);
        document.body.style.overflow = 'hidden';

        return () =>
        {
            document.removeEventListener('keydown', closeOnEscape);
            document.body.style.overflow = '';
            previouslyFocused?.focus();
        };
    }, [isOpen]);

    if (!isOpen) return null;

    return createPortal(
        <div className={styles.overlay} onMouseDown={event => event.target === event.currentTarget && onClose()}>
            <div ref={dialogRef} className={`${styles.dialog} ${styles[size]}`} role='dialog' aria-modal='true' aria-labelledby={titleId}>
                <header className={styles.header}>
                    {icon && <span className={styles.icon}>{icon}</span>}
                    <div className={styles.headerText}>
                        <h2 id={titleId} className={styles.title}>{title}</h2>
                        {description && <p className={styles.description}>{description}</p>}
                    </div>
                    <button type='button' className={styles.close} onClick={onClose} aria-label='Cerrar' data-close>
                        <X size={18}/>
                    </button>
                </header>
                <div className={styles.body}>{children}</div>
                {footer && <footer className={styles.footer}>{footer}</footer>}
            </div>
        </div>,
        document.body
    );
}

export default Modal;
