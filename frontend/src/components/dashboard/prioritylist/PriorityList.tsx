import styles from './prioritylist.module.css';
import {Link} from 'react-router';
import {ArrowRight} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import type {AnomalySummary} from '@/interfaces/interfaces.ts';
import {formatNumber, formatPercent} from '@/lib/format.ts';
import {ANOMALY_TYPE, SEVERITY} from '@/lib/labels.ts';

interface PriorityListProps {
    anomalies: AnomalySummary[];
}

const PriorityList = ({anomalies}: PriorityListProps) =>
{
    return (
        <ol className={styles.list}>
            {anomalies.map((anomaly, index) =>
            {
                const type = ANOMALY_TYPE[anomaly.type];
                const severity = SEVERITY[anomaly.severity];

                return (
                    <li key={anomaly.id} className={styles.item}>
                        <div className={styles.row}>
                            <span className={`${styles.rank} ${styles[type.tone]}`}>#{index + 1}</span>
                            <span className={`${styles.code} mono`}>{anomaly.meterCode}</span>
                            <Badge tone={type.tone}>{type.label}</Badge>
                            <Badge tone={severity.tone}>{severity.label}</Badge>
                            {anomaly.variationPct !== null && (
                                <span className={`${styles.variation} tabular`}>{formatPercent(anomaly.variationPct)}</span>
                            )}
                            <span className={styles.score}>
                                <span className={styles.scoreLabel}>Prioridad</span>
                                <strong className='tabular'>{formatNumber(anomaly.priorityScore, 1)}</strong>
                            </span>
                        </div>

                        <div className={styles.track} aria-hidden='true'>
                            <div className={`${styles.bar} ${styles[type.tone]}`} style={{width: `${Math.max(anomaly.priorityScore, 2)}%`}}/>
                        </div>

                        <div className={styles.row}>
                            <p className={styles.reason}>{anomaly.reason}</p>
                            <Link to={`/anomalies/${anomaly.id}`} className={styles.link}>
                                Investigar <ArrowRight size={14}/>
                            </Link>
                        </div>
                    </li>
                );
            })}
        </ol>
    );
}

export default PriorityList;
