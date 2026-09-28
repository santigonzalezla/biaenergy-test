import styles from './runanalysisbutton.module.css';
import {Loader2, Sparkles} from 'lucide-react';
import {useAnalysis} from '@/context/analysis.ts';

interface RunAnalysisButtonProps {
    size?: 'sm' | 'md';
    meterIds?: string[];
    label?: string;
}

const RunAnalysisButton = ({size = 'md', meterIds, label = 'Ejecutar análisis IA'}: RunAnalysisButtonProps) =>
{
    const {analysis, isActive, isStarting, start, openDrawer} = useAnalysis();
    const busy = isActive || isStarting;

    const handleClick = () =>
    {
        if (busy)
        {
            openDrawer();
            return;
        }

        void start(meterIds);
    }

    return (
        <button type='button' className={`${styles.button} ${styles[size]}`} onClick={handleClick} aria-busy={busy}>
            {busy
                ? <><Loader2 size={16} className={styles.spinner}/> Analizando… {analysis && isActive ? `${analysis.progress} %` : ''}</>
                : <><Sparkles size={16}/> {label}</>}
        </button>
    );
}

export default RunAnalysisButton;
