import styles from './anomalydetail.module.css';
import {useState} from 'react';
import {Link, useParams} from 'react-router';
import {ArrowLeft, ArrowRight, Lightbulb, SearchX, Sparkles} from 'lucide-react';
import EvidencePanel from '@/components/investigation/evidencepanel/EvidencePanel.tsx';
import RelatedEvent from '@/components/investigation/relatedevent/RelatedEvent.tsx';
import StatusWorkflow from '@/components/investigation/statusworkflow/StatusWorkflow.tsx';
import ConsumptionChart from '@/components/meterdetail/consumptionchart/ConsumptionChart.tsx';
import Badge from '@/components/shared/badge/Badge.tsx';
import Card from '@/components/shared/card/Card.tsx';
import Skeleton from '@/components/shared/skeleton/Skeleton.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {useFetch} from '@/hooks/useFetch.ts';
import type {AnomalyDetail, Reading, Series} from '@/interfaces/interfaces.ts';
import {buildQuery} from '@/lib/api.ts';
import {formatDateTime, formatNumber, formatPercent, formatRatio} from '@/lib/format.ts';
import {ANOMALY_STATUS, ANOMALY_TYPE, SEVERITY} from '@/lib/labels.ts';

const DAY_MS = 24 * 60 * 60 * 1000;

const chartWindow = (anomaly: AnomalyDetail) =>
{
    const start = Date.parse(anomaly.windowStart ?? anomaly.detectedAt);
    const end = Date.parse(anomaly.windowEnd ?? anomaly.windowStart ?? anomaly.detectedAt);

    return {from: new Date(start - 4 * DAY_MS).toISOString(), to: new Date(end + 3 * DAY_MS).toISOString()};
}

const AnomalyDetailPage = () =>
{
    const {anomalyId = ''} = useParams();
    const detail = useFetch<AnomalyDetail>(`/anomalies/${anomalyId}`);
    const [updated, setUpdated] = useState<AnomalyDetail | null>(null);

    const anomaly = updated?.id === anomalyId ? updated : detail.data;
    const readings = useFetch<Series<Reading>>(anomaly ? `/meters/${anomaly.meter.id}/readings${buildQuery(chartWindow(anomaly))}` : null);

    if (detail.error && !detail.data)
    {
        const notFound = detail.error.status === 404 || detail.error.status === 400;

        return (
            <Card>
                <StateMessage
                    variant={notFound ? 'empty' : 'error'}
                    icon={notFound ? SearchX : undefined}
                    title={notFound ? 'Anomalía no encontrada' : 'No se pudo cargar la anomalía'}
                    description={detail.error.message}
                    onRetry={notFound ? undefined : () => void detail.execute()}
                    action={<Link to='/anomalies' className={styles.back}><ArrowLeft size={14}/> Volver a anomalías</Link>}
                />
            </Card>
        );
    }

    if (!anomaly)
    {
        return (
            <div className={styles.page}>
                <Skeleton height='96px' radius='var(--radius-lg)'/>
                <Skeleton height='320px' radius='var(--radius-lg)'/>
            </div>
        );
    }

    const type = ANOMALY_TYPE[anomaly.type];
    const severity = SEVERITY[anomaly.severity];
    const status = ANOMALY_STATUS[anomaly.status];
    const changeAt = anomaly.windowStart ?? anomaly.detectedAt;
    const relatedEvent = anomaly.relatedEvent ?? anomaly.evidence.relatedEvent;

    return (
        <div className={styles.page}>
            <header className={styles.header}>
                <Link to='/anomalies' className={styles.back}><ArrowLeft size={14}/> Anomalías IA</Link>

                <div className={styles.titleRow}>
                    <h1 className={styles.title}>
                        <span className='mono'>{anomaly.meter.code}</span>
                        <span className={styles.titleType}>{type.label}</span>
                    </h1>
                    <Badge tone={type.tone}>{type.label}</Badge>
                    <Badge tone={severity.tone}>Severidad {severity.label.toLowerCase()}</Badge>
                    <Badge tone={status.tone} dot>{status.label}</Badge>
                    <Link to={`/meters/${anomaly.meter.id}`} className={styles.meterLink}>
                        Ver medidor <ArrowRight size={14}/>
                    </Link>
                </div>

                <dl className={styles.metrics}>
                    <div>
                        <dt>Prioridad</dt>
                        <dd className='tabular'>{formatNumber(anomaly.priorityScore, 1)}</dd>
                    </div>
                    <div>
                        <dt>Confianza</dt>
                        <dd className='tabular'>{formatRatio(anomaly.confidence)}</dd>
                    </div>
                    <div>
                        <dt>Consumo</dt>
                        <dd className='tabular'>
                            {anomaly.baselineKwh !== null && anomaly.currentKwh !== null
                                ? `${formatNumber(anomaly.baselineKwh, 1)} → ${formatNumber(anomaly.currentKwh, 1)} kWh/h`
                                : '—'}
                        </dd>
                    </div>
                    <div>
                        <dt>Variación</dt>
                        <dd className='tabular'>{anomaly.variationPct === null ? '—' : formatPercent(anomaly.variationPct)}</dd>
                    </div>
                    <div>
                        <dt>Desde</dt>
                        <dd>{formatDateTime(changeAt)}</dd>
                    </div>
                </dl>
            </header>

            <div className={styles.columns}>
                <div className={styles.main}>
                    <section className={styles.explanation} aria-labelledby='explanation-title'>
                        <h2 id='explanation-title' className={styles.explanationTitle}>
                            <Sparkles size={18}/> Explicación IA
                        </h2>
                        <p className={styles.explanationText}>{anomaly.reason}</p>
                        <span className={styles.rule}>Regla aplicada: <span className='mono'>{anomaly.ruleId}</span></span>
                    </section>

                    <Card title='Ventana de la anomalía' subtitle='Consumo horario alrededor del cambio, con su línea base'>
                        {!readings.data && !readings.error && <Skeleton height='300px'/>}
                        {readings.error && <StateMessage variant='error' title='No se pudo cargar el consumo' onRetry={() => void readings.execute()}/>}
                        {readings.data && <ConsumptionChart readings={readings.data.data} baselineKwh={anomaly.baselineKwh} changeAt={changeAt}/>}
                    </Card>

                    <Card title='Evidencia' subtitle='Lo que midió el motor de detección antes y después del cambio'>
                        <EvidencePanel evidence={anomaly.evidence}/>
                    </Card>
                </div>

                <aside className={styles.side}>
                    <section className={styles.action} aria-labelledby='action-title'>
                        <h2 id='action-title' className={styles.actionTitle}><Lightbulb size={18}/> Acción recomendada</h2>
                        <p>{anomaly.recommendedAction}</p>
                    </section>

                    <Card title='Seguimiento'>
                        <StatusWorkflow anomaly={anomaly} onUpdated={setUpdated}/>
                    </Card>

                    <Card title='Evento relacionado' subtitle='Evento operativo en la ventana de ±6 horas'>
                        <RelatedEvent event={relatedEvent}/>
                    </Card>
                </aside>
            </div>
        </div>
    );
}

export default AnomalyDetailPage;
