import styles from './alertmeters.module.css';
import {Link} from 'react-router';
import {ArrowRight, ShieldCheck} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {MeterStatus} from '@/interfaces/enums.ts';
import type {Meter} from '@/interfaces/interfaces.ts';
import {formatPercent} from '@/lib/format.ts';
import {METER_STATUS} from '@/lib/labels.ts';

interface AlertMetersProps {
    meters: Meter[];
}

const SEVERITY_ORDER: Record<MeterStatus, number> = {CRITICAL: 0, ALERT: 1, OK: 2};

const AlertMeters = ({meters}: AlertMetersProps) =>
{
    const alerting = meters
        .filter(meter => meter.status !== MeterStatus.OK)
        .sort((a, b) => SEVERITY_ORDER[a.status] - SEVERITY_ORDER[b.status]);

    if (alerting.length === 0)
    {
        return <StateMessage icon={ShieldCheck} title='Sin medidores en alerta' description='Todos los medidores están dentro de lo normal según el último análisis.'/>;
    }

    return (
        <ul className={styles.list}>
            {alerting.map(meter =>
            {
                const status = METER_STATUS[meter.status];

                return (
                    <li key={meter.id} className={`${styles.item} ${styles[status.tone]}`}>
                        <div className={styles.top}>
                            <span className={`${styles.code} mono`}>{meter.code}</span>
                            <Badge tone={status.tone} dot pulse={meter.status === MeterStatus.CRITICAL}>{status.label}</Badge>
                            {meter.stats && (
                                <span className={`${styles.variation} tabular`}>{formatPercent(meter.stats.variationPct)} vs. línea base</span>
                            )}
                        </div>
                        <div className={styles.bottom}>
                            <span className={styles.name}>{meter.name}</span>
                            <Link to={`/meters/${meter.id}`} className={styles.link}>
                                Ver medidor <ArrowRight size={14}/>
                            </Link>
                        </div>
                    </li>
                );
            })}
        </ul>
    );
}

export default AlertMeters;
