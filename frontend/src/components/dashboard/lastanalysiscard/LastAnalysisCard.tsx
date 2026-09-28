import styles from './lastanalysiscard.module.css';
import {AlertTriangle, CalendarCheck, Gauge, Sparkles} from 'lucide-react';
import RunAnalysisButton from '@/components/analysis/runanalysisbutton/RunAnalysisButton.tsx';
import Badge from '@/components/shared/badge/Badge.tsx';
import {useAnalysis} from '@/context/analysis.ts';
import {formatDateTime} from '@/lib/format.ts';
import {ANALYSIS_STATUS, ANALYSIS_STEP} from '@/lib/labels.ts';

const LastAnalysisCard = () =>
{
    const {analysis, isActive, openDrawer} = useAnalysis();

    if (!analysis)
    {
        return (
            <section className={`${styles.card} ${styles.empty}`}>
                <div className={styles.emptyIcon}>
                    <Sparkles size={28}/>
                </div>
                <h2 className={styles.title}>Aún no hay análisis</h2>
                <p className={styles.emptyText}>
                    Ejecuta el análisis IA para evaluar las lecturas de todos los medidores y detectar anomalías.
                </p>
                <RunAnalysisButton/>
            </section>
        );
    }

    const status = ANALYSIS_STATUS[analysis.status];
    const timestamp = analysis.finishedAt ?? analysis.createdAt;

    return (
        <section className={styles.card}>
            <div className={styles.info}>
                <div className={styles.heading}>
                    <h2 className={styles.title}>Último análisis IA</h2>
                    <Badge tone={status.tone} dot pulse={isActive}>
                        {isActive && analysis.currentStep ? `${ANALYSIS_STEP[analysis.currentStep]} · ${analysis.progress} %` : status.label}
                    </Badge>
                </div>

                <ul className={styles.chips}>
                    <li className={styles.chip}>
                        <CalendarCheck size={14}/>
                        {analysis.finishedAt ? 'Finalizado' : 'Iniciado'} {formatDateTime(timestamp)}
                    </li>
                    {!isActive && (
                        <>
                            <li className={styles.chip}>
                                <Gauge size={14}/>
                                {analysis.metersAnalyzed} medidores analizados
                            </li>
                            <li className={styles.chip}>
                                <Sparkles size={14}/>
                                {analysis.anomalies} anomalías
                            </li>
                            <li className={`${styles.chip} ${styles.danger}`}>
                                <AlertTriangle size={14}/>
                                {analysis.highPriority} de alta prioridad
                            </li>
                        </>
                    )}
                </ul>
            </div>

            <div className={styles.actions}>
                {isActive && (
                    <button type='button' className={styles.link} onClick={openDrawer}>Ver progreso</button>
                )}
                <RunAnalysisButton/>
            </div>
        </section>
    );
}

export default LastAnalysisCard;
