import styles from './meterdetail.module.css';
import {useEffect, useState} from 'react';
import {Link, useParams} from 'react-router';
import {ArrowLeft, Clock, Gauge, MapPin, Plug, SearchX, Zap} from 'lucide-react';
import AnomalyCallout from '@/components/meterdetail/anomalycallout/AnomalyCallout.tsx';
import ConsumptionChart from '@/components/meterdetail/consumptionchart/ConsumptionChart.tsx';
import ElectricalPanel from '@/components/meterdetail/electricalpanel/ElectricalPanel.tsx';
import EventTimeline from '@/components/meterdetail/eventtimeline/EventTimeline.tsx';
import HourlyProfileChart from '@/components/meterdetail/hourlyprofilechart/HourlyProfileChart.tsx';
import Badge from '@/components/shared/badge/Badge.tsx';
import Card from '@/components/shared/card/Card.tsx';
import Skeleton from '@/components/shared/skeleton/Skeleton.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {useAnalysis} from '@/context/analysis.ts';
import {useFetch} from '@/hooks/useFetch.ts';
import {MeterStatus} from '@/interfaces/enums.ts';
import type {AnomalyDetail, AnomalyList, HourlyProfile, Meter, MeterEvent, Reading, Series} from '@/interfaces/interfaces.ts';
import {buildQuery} from '@/lib/api.ts';
import {formatDayMonth, formatNumber, formatPercent, formatShortDateTime} from '@/lib/format.ts';
import {ANOMALY_TYPE, METER_STATUS, SEVERITY} from '@/lib/labels.ts';
import {rangeEndingAt} from '@/lib/series.ts';

const RANGE_OPTIONS = [3, 7, 14];
const FULL_RANGE_DAYS = 14;

const MeterDetailPage = () =>
{
    const {meterId = ''} = useParams();
    const {completedVersion} = useAnalysis();
    const [days, setDays] = useState(FULL_RANGE_DAYS);

    const meter = useFetch<Meter>(`/meters/${meterId}`);
    const anomalies = useFetch<AnomalyList>('/anomalies');

    const meterAnomalies = (anomalies.data?.data ?? []).filter(anomaly => anomaly.meterId === meterId);
    const main = meterAnomalies[0] ?? null;
    const detail = useFetch<AnomalyDetail>(main ? `/anomalies/${main.id}` : null);

    const lastReadingAt = meter.data?.stats?.lastReadingAt ?? null;
    const range = lastReadingAt ? rangeEndingAt(lastReadingAt, days) : null;
    const fullRange = lastReadingAt ? rangeEndingAt(lastReadingAt, FULL_RANGE_DAYS) : null;
    const changeAt = detail.data ? detail.data.windowStart ?? detail.data.detectedAt : null;

    const readings = useFetch<Series<Reading>>(range ? `/meters/${meterId}/readings${buildQuery({...range})}` : null);
    const events = useFetch<Series<MeterEvent>>(fullRange ? `/meters/${meterId}/events${buildQuery({...fullRange})}` : null);
    const profileBefore = useFetch<HourlyProfile>(fullRange ? `/meters/${meterId}/profile${buildQuery({from: fullRange.from, to: changeAt ?? fullRange.to})}` : null);
    const profileAfter = useFetch<HourlyProfile>(fullRange && changeAt ? `/meters/${meterId}/profile${buildQuery({from: changeAt, to: fullRange.to})}` : null);

    const {execute: refreshMeter} = meter;
    const {execute: refreshAnomalies} = anomalies;

    useEffect(() =>
    {
        if (completedVersion === 0) return;

        void refreshMeter();
        void refreshAnomalies();
    }, [completedVersion, refreshMeter, refreshAnomalies]);

    if (meter.error && !meter.data)
    {
        const notFound = meter.error.status === 404 || meter.error.status === 400;

        return (
            <Card>
                <StateMessage
                    variant={notFound ? 'empty' : 'error'}
                    icon={notFound ? SearchX : undefined}
                    title={notFound ? 'Medidor no encontrado' : 'No se pudo cargar el medidor'}
                    description={meter.error.message}
                    onRetry={notFound ? undefined : () => void refreshMeter()}
                    action={<Link to='/meters' className={styles.back}><ArrowLeft size={14}/> Volver a medidores</Link>}
                />
            </Card>
        );
    }

    if (!meter.data)
    {
        return (
            <div className={styles.page}>
                <Skeleton height='64px' radius='var(--radius-lg)'/>
                <Skeleton height='360px' radius='var(--radius-lg)'/>
            </div>
        );
    }

    const data = meter.data;
    const status = METER_STATUS[data.status];
    const baselineKwh = detail.data?.baselineKwh ?? data.stats?.baselineKwh ?? null;
    const recentKwh = detail.data?.currentKwh ?? data.stats?.recentKwh ?? null;
    const variation = main?.variationPct ?? data.stats?.variationPct ?? null;

    return (
        <div className={styles.page}>
            <header className={styles.header}>
                <Link to='/meters' className={styles.back}><ArrowLeft size={14}/> Medidores</Link>
                <div className={styles.titleRow}>
                    <span className={`${styles.code} mono`}>{data.code}</span>
                    <h1 className={styles.title}>{data.name}</h1>
                    <Badge tone={status.tone} dot pulse={data.status === MeterStatus.CRITICAL}>{status.label}</Badge>
                </div>
                <ul className={styles.specs}>
                    {data.location && <li><MapPin size={14}/>{data.location}</li>}
                    <li><Gauge size={14}/>Sector {data.sector || '—'}</li>
                    <li><Plug size={14}/>Voltaje nominal <strong className='tabular'>{formatNumber(data.nominalVoltage)} V</strong></li>
                    {data.contractedPowerKw !== null && <li><Zap size={14}/>Potencia contratada <strong className='tabular'>{formatNumber(data.contractedPowerKw)} kW</strong></li>}
                    {lastReadingAt && <li><Clock size={14}/>Última lectura <strong>{formatShortDateTime(lastReadingAt)}</strong></li>}
                </ul>
            </header>

            {main && <AnomalyCallout anomaly={main} changeAt={changeAt}/>}

            <div className={styles.columns}>
                <div className={styles.main}>
                    <Card
                        title='Consumo horario'
                        subtitle={range ? `${formatDayMonth(range.from)} – ${formatDayMonth(lastReadingAt!)} · hora de Bogotá` : undefined}
                        actions={
                            <div className={styles.ranges} role='group' aria-label='Rango de días'>
                                {RANGE_OPTIONS.map(option => (
                                    <button
                                        key={option}
                                        type='button'
                                        className={`${styles.range} ${days === option ? styles.rangeActive : ''}`}
                                        aria-pressed={days === option}
                                        onClick={() => setDays(option)}
                                    >
                                        {option} días
                                    </button>
                                ))}
                            </div>
                        }
                    >
                        <div className={styles.kpis}>
                            <div className={styles.kpi}>
                                <span>Línea base</span>
                                <strong className='tabular'>{baselineKwh === null ? '—' : `${formatNumber(baselineKwh, 1)} kWh/h`}</strong>
                            </div>
                            <div className={`${styles.kpi} ${main ? styles[ANOMALY_TYPE[main.type].tone] : ''}`}>
                                <span>{main ? 'Nivel desde el cambio' : 'Últimas 48 h'}</span>
                                <strong className='tabular'>{recentKwh === null ? '—' : `${formatNumber(recentKwh, 1)} kWh/h`}</strong>
                            </div>
                            <div className={styles.kpi}>
                                <span>Variación</span>
                                <strong className='tabular'>{variation === null ? '—' : formatPercent(variation)}</strong>
                            </div>
                            <div className={styles.kpi}>
                                <span>Factor de potencia mínimo</span>
                                <strong className='tabular'>{data.stats ? formatNumber(data.stats.minPowerFactor, 2) : '—'}</strong>
                            </div>
                        </div>

                        {!lastReadingAt && <StateMessage icon={Gauge} title='Sin lecturas' description='Importa lecturas para ver el consumo de este medidor.'/>}
                        {lastReadingAt && !readings.data && <Skeleton height='300px'/>}
                        {readings.data && <ConsumptionChart readings={readings.data.data} baselineKwh={baselineKwh} changeAt={changeAt}/>}
                    </Card>

                    {readings.data && readings.data.data.length > 0 && (
                        <Card title='Variables eléctricas' subtitle='Cada variable con el umbral que usa el motor de detección'>
                            <ElectricalPanel readings={readings.data.data} nominalVoltage={data.nominalVoltage} changeAt={changeAt}/>
                        </Card>
                    )}

                    {profileBefore.data && (
                        <Card
                            title='Perfil horario promedio'
                            subtitle={changeAt ? 'Consumo promedio por hora del día, antes y después del punto de cambio' : 'Consumo promedio por hora del día'}
                        >
                            <HourlyProfileChart
                                before={profileBefore.data}
                                after={profileAfter.data}
                                beforeLabel={changeAt ? `Antes (hasta ${formatDayMonth(changeAt)})` : 'Promedio'}
                                afterLabel={changeAt ? `Después (desde ${formatDayMonth(changeAt)})` : ''}
                            />
                        </Card>
                    )}
                </div>

                <aside className={styles.side}>
                    <Card title='Eventos operativos' subtitle={`Últimos ${FULL_RANGE_DAYS} días de datos`}>
                        {!events.data && lastReadingAt && <Skeleton height='80px'/>}
                        {events.data && <EventTimeline events={events.data.data} changeAt={changeAt}/>}
                        {!lastReadingAt && <StateMessage title='Sin periodo de datos'/>}
                    </Card>

                    <Card title='Anomalías del último análisis'>
                        {meterAnomalies.length === 0
                            ? <StateMessage icon={Gauge} title='Sin anomalías' description='El último análisis no encontró anomalías en este medidor.'/>
                            : (
                                <ul className={styles.anomalies}>
                                    {meterAnomalies.map(anomaly => (
                                        <li key={anomaly.id}>
                                            <Link to={`/anomalies/${anomaly.id}`} className={styles.anomaly}>
                                                <div className={styles.anomalyBadges}>
                                                    <Badge tone={ANOMALY_TYPE[anomaly.type].tone}>{ANOMALY_TYPE[anomaly.type].label}</Badge>
                                                    <Badge tone={SEVERITY[anomaly.severity].tone}>{SEVERITY[anomaly.severity].label}</Badge>
                                                </div>
                                                <span className={styles.anomalyMeta}>Prioridad {formatNumber(anomaly.priorityScore, 1)} · {formatShortDateTime(anomaly.detectedAt)}</span>
                                            </Link>
                                        </li>
                                    ))}
                                </ul>
                            )}
                    </Card>
                </aside>
            </div>
        </div>
    );
}

export default MeterDetailPage;
