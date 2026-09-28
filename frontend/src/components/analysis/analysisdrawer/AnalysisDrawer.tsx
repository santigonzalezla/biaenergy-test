import styles from './analysisdrawer.module.css';
import {useEffect, useState} from 'react';
import {useNavigate} from 'react-router';
import {AlertTriangle, Check, CheckCircle2, Clock, Info, Loader2, RotateCw, Sparkles, X} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import {useAnalysis} from '@/context/analysis.ts';
import {AnalysisStatus, AnalysisStep} from '@/interfaces/enums.ts';
import type {Analysis} from '@/interfaces/interfaces.ts';
import {formatDuration, formatShortDateTime} from '@/lib/format.ts';
import {ANALYSIS_STATUS, ANALYSIS_STEP} from '@/lib/labels.ts';

type StepState = 'done' | 'current' | 'failed' | 'pending';

const STEPS: AnalysisStep[] = [AnalysisStep.LOADING_DATA, AnalysisStep.ANALYZING, AnalysisStep.SAVING, AnalysisStep.COMPLETED];

const stepStateOf = (analysis: Analysis, index: number): StepState =>
{
    if (analysis.status === AnalysisStatus.COMPLETED) return 'done';

    const current = analysis.currentStep ? STEPS.indexOf(analysis.currentStep) : -1;

    if (index < current) return 'done';
    if (index === current) return analysis.status === AnalysisStatus.FAILED ? 'failed' : 'current';

    return 'pending';
}

const elapsedOf = (analysis: Analysis, now: number) =>
{
    const start = Date.parse(analysis.startedAt ?? analysis.createdAt);
    const end = analysis.finishedAt ? Date.parse(analysis.finishedAt) : now;

    return end - start;
}

const AnalysisDrawer = () =>
{
    const {analysis, isActive, isStarting, isDrawerOpen, closeDrawer, start} = useAnalysis();
    const navigate = useNavigate();
    const [now, setNow] = useState(() => Date.now());

    useEffect(() =>
    {
        if (!isDrawerOpen || !isActive) return;

        const interval = setInterval(() => setNow(Date.now()), 1000);

        return () => clearInterval(interval);
    }, [isDrawerOpen, isActive]);

    useEffect(() =>
    {
        if (!isDrawerOpen) return;

        const closeOnEscape = (event: KeyboardEvent) =>
        {
            if (event.key === 'Escape') closeDrawer();
        };

        document.addEventListener('keydown', closeOnEscape);

        return () => document.removeEventListener('keydown', closeOnEscape);
    }, [isDrawerOpen, closeDrawer]);

    const goToAnomalies = () =>
    {
        closeDrawer();
        navigate('/anomalies');
    }

    const isCompleted = analysis?.status === AnalysisStatus.COMPLETED;
    const isFailed = analysis?.status === AnalysisStatus.FAILED;
    const status = analysis ? ANALYSIS_STATUS[analysis.status] : null;

    return (
        <>
            <div className={`${styles.backdrop} ${isDrawerOpen ? styles.open : ''}`} onClick={closeDrawer} aria-hidden='true'/>

            <aside
                className={`${styles.drawer} ${isDrawerOpen ? styles.open : ''}`}
                role='dialog'
                aria-modal='true'
                aria-labelledby='analysis-drawer-title'
                aria-hidden={!isDrawerOpen}
            >
                <header className={styles.header}>
                    <div className={styles.headerIcon}>
                        <Sparkles size={20}/>
                    </div>
                    <div className={styles.headerText}>
                        <h2 id='analysis-drawer-title' className={styles.title}>Progreso del análisis IA</h2>
                        <p className={styles.subtitle}>Detección, clasificación y explicación de anomalías</p>
                    </div>
                    <button type='button' className={styles.close} onClick={closeDrawer} aria-label='Cerrar panel'>
                        <X size={18}/>
                    </button>
                </header>

                <div className={styles.body}>
                    {!analysis && (
                        <div className={styles.placeholder}>
                            {isStarting ? <><Loader2 size={20} className={styles.spinner}/> Iniciando análisis…</> : 'Aún no se ha ejecutado ningún análisis.'}
                        </div>
                    )}

                    {analysis && status && (
                        <>
                            <div className={styles.summary}>
                                <div className={styles.summaryTop}>
                                    <Badge tone={status.tone} dot pulse={isActive}>{status.label}</Badge>
                                    <span className={styles.elapsed}>
                                        <Clock size={14}/>
                                        {isActive ? 'Transcurrido' : 'Duración'} <strong className='tabular'>{formatDuration(elapsedOf(analysis, now))}</strong>
                                    </span>
                                </div>

                                <div className={styles.progressValue}>
                                    <span className={`${styles.percent} tabular ${isFailed ? styles.failedText : ''}`}>{analysis.progress} %</span>
                                    <span className={styles.analysisId}>Análisis #{analysis.numId} · {formatShortDateTime(analysis.createdAt)}</span>
                                </div>

                                <div
                                    className={styles.progressTrack}
                                    role='progressbar'
                                    aria-valuemin={0}
                                    aria-valuemax={100}
                                    aria-valuenow={analysis.progress}
                                >
                                    <div
                                        className={`${styles.progressBar} ${isFailed ? styles.failedBar : ''} ${isActive ? styles.animated : ''}`}
                                        style={{width: `${analysis.progress}%`}}
                                    />
                                </div>
                            </div>

                            {isActive && (
                                <div className={styles.note}>
                                    <Info size={16}/>
                                    <p>Puedes cerrar este panel y seguir navegando. Te avisaremos cuando termine.</p>
                                </div>
                            )}

                            <ol className={styles.steps}>
                                {STEPS.map((step, index) =>
                                {
                                    const state = stepStateOf(analysis, index);

                                    return (
                                        <li key={step} className={`${styles.step} ${styles[state]}`}>
                                            <span className={styles.stepNode}>
                                                {state === 'done' && <Check size={14}/>}
                                                {state === 'current' && <Loader2 size={14} className={styles.spinner}/>}
                                                {state === 'failed' && <X size={14}/>}
                                                {state === 'pending' && index + 1}
                                            </span>
                                            <span className={styles.stepLabel}>{ANALYSIS_STEP[step]}</span>
                                        </li>
                                    );
                                })}
                            </ol>

                            {isCompleted && (
                                <div className={styles.results}>
                                    <div className={styles.result}>
                                        <span>Medidores analizados</span>
                                        <strong className='tabular'>{analysis.metersAnalyzed}</strong>
                                    </div>
                                    <div className={styles.result}>
                                        <span>Anomalías detectadas</span>
                                        <strong className='tabular'>{analysis.anomalies}</strong>
                                    </div>
                                    <div className={`${styles.result} ${styles.highlight}`}>
                                        <span>Alta prioridad</span>
                                        <strong className='tabular'>{analysis.highPriority}</strong>
                                    </div>
                                </div>
                            )}

                            {isFailed && (
                                <div className={styles.failure} role='alert'>
                                    <AlertTriangle size={18}/>
                                    <div>
                                        <strong>El análisis no pudo completarse</strong>
                                        <p>{analysis.error ?? 'El servicio de IA no respondió.'}</p>
                                    </div>
                                </div>
                            )}
                        </>
                    )}
                </div>

                <footer className={styles.footer}>
                    {isCompleted && (
                        <button type='button' className={styles.primary} onClick={goToAnomalies}>
                            <CheckCircle2 size={16}/>
                            Ver anomalías
                        </button>
                    )}
                    {isFailed && (
                        <button type='button' className={styles.primary} onClick={() => void start()}>
                            <RotateCw size={16}/>
                            Reintentar
                        </button>
                    )}
                    <button type='button' className={styles.secondary} onClick={closeDrawer}>
                        {isActive ? 'Seguir navegando' : 'Cerrar'}
                    </button>
                </footer>
            </aside>
        </>
    );
}

export default AnalysisDrawer;
