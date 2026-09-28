import styles from './anomalycard.module.css';
import {Link} from 'react-router';
import {ArrowRight} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import type {AnomalySummary} from '@/interfaces/interfaces.ts';
import {formatNumber, formatPercent, formatRatio, formatShortDateTime} from '@/lib/format.ts';
import {ANOMALY_STATUS, ANOMALY_TYPE, SEVERITY} from '@/lib/labels.ts';

interface AnomalyCardProps {
    anomaly: AnomalySummary;
    rank: number;
}

const AnomalyCard = ({anomaly, rank}: AnomalyCardProps) =>
{
    const type = ANOMALY_TYPE[anomaly.type];
    const severity = SEVERITY[anomaly.severity];
    const status = ANOMALY_STATUS[anomaly.status];

    return (
        <article className={`${styles.card} ${styles[type.tone]}`} aria-labelledby={`anomaly-${anomaly.id}`}>
            <div className={styles.rank}>#{rank}</div>

            <div className={styles.body}>
                <header className={styles.header}>
                    <h2 id={`anomaly-${anomaly.id}`} className={styles.meter}>
                        <span className='mono'>{anomaly.meterCode}</span>
                        <span className={styles.meterName}>{anomaly.meterName}</span>
                    </h2>
                    <div className={styles.badges}>
                        <Badge tone={type.tone}>{type.label}</Badge>
                        <Badge tone={severity.tone}>Severidad {severity.label.toLowerCase()}</Badge>
                        <Badge tone={status.tone} dot>{status.label}</Badge>
                    </div>
                </header>

                <p className={styles.reason}>{anomaly.reason}</p>

                <dl className={styles.metrics}>
                    <div>
                        <dt>Prioridad</dt>
                        <dd className='tabular'>{formatNumber(anomaly.priorityScore, 1)}</dd>
                    </div>
                    <div>
                        <dt>Confianza</dt>
                        <dd>
                            <span className={styles.confidence} aria-hidden='true'>
                                <span style={{width: `${anomaly.confidence * 100}%`}}/>
                            </span>
                            <span className='tabular'>{formatRatio(anomaly.confidence)}</span>
                        </dd>
                    </div>
                    <div>
                        <dt>Variación</dt>
                        <dd className='tabular'>{anomaly.variationPct === null ? '—' : formatPercent(anomaly.variationPct)}</dd>
                    </div>
                    <div>
                        <dt>Detectada</dt>
                        <dd>{formatShortDateTime(anomaly.detectedAt)}</dd>
                    </div>
                </dl>
            </div>

            <Link to={`/anomalies/${anomaly.id}`} className={styles.link} aria-label={`Investigar anomalía de ${anomaly.meterCode}`}>
                Investigar <ArrowRight size={16}/>
            </Link>
        </article>
    );
}

export default AnomalyCard;
