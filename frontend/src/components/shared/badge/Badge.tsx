import styles from './badge.module.css';
import type {ReactNode} from 'react';
import type {Tone} from '@/lib/labels.ts';

interface BadgeProps {
    tone: Tone;
    children: ReactNode;
    dot?: boolean;
    pulse?: boolean;
}

const Badge = ({tone, children, dot = false, pulse = false}: BadgeProps) =>
{
    return (
        <span className={`${styles.badge} ${styles[tone]}`}>
            {dot && <span className={`${styles.dot} ${pulse ? styles.pulse : ''}`}/>}
            {children}
        </span>
    );
}

export default Badge;
