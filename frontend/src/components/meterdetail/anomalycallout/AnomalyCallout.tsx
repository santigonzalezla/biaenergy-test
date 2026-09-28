import styles from './anomalycallout.module.css';
import {Link} from 'react-router';
import {AlertOctagon, ArrowRight} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import type {AnomalySummary} from '@/interfaces/interfaces.ts';
import {formatDateTime, formatPercent} from '@/lib/format.ts';
import {ANOMALY_TYPE, SEVERITY} from '@/lib/labels.ts';

interface AnomalyCalloutProps {
    anomaly: AnomalySummary;
    changeAt: string | null;
}

const AnomalyCallout = ({anomaly, changeAt}: AnomalyCalloutProps) =>
{
    const type = ANOMALY_TYPE[anomaly.type];
    const severity = SEVERITY[anomaly.severity];

    return (
        <section className={`${styles.callout} ${styles[type.tone]}`} aria-label='Anomalía detectada en el último análisis'>
            <div className={styles.icon}>
                <AlertOctagon size={22}/>
            </div>
            <div className={styles.content}>
                <div className={styles.heading}>
                    <Badge tone={type.tone}>{type.label}</Badge>
                    <Badge tone={severity.tone}>Severidad {severity.label.toLowerCase()}</Badge>
                    {anomaly.variationPct !== null && <strong className='tabular'>{formatPercent(anomaly.variationPct)} de consumo</strong>}
                    {changeAt && <span className={styles.since}>desde {formatDateTime(changeAt)}</span>}
                </div>
                <p className={styles.reason}>{anomaly.reason}</p>
            </div>
            <Link to={`/anomalies/${anomaly.id}`} className={styles.link}>
                Ver investigación <ArrowRight size={16}/>
            </Link>
        </section>
    );
}

export default AnomalyCallout;
