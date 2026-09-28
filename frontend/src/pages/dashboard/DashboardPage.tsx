import styles from './dashboard.module.css';
import {useEffect} from 'react';
import {Activity, AlertTriangle, Gauge, ListChecks, Zap} from 'lucide-react';
import AlertMeters from '@/components/dashboard/alertmeters/AlertMeters.tsx';
import KpiCard from '@/components/dashboard/kpicard/KpiCard.tsx';
import LastAnalysisCard from '@/components/dashboard/lastanalysiscard/LastAnalysisCard.tsx';
import PriorityList from '@/components/dashboard/prioritylist/PriorityList.tsx';
import VariationChart from '@/components/dashboard/variationchart/VariationChart.tsx';
import Badge from '@/components/shared/badge/Badge.tsx';
import Card from '@/components/shared/card/Card.tsx';
import PageHeader from '@/components/shared/pageheader/PageHeader.tsx';
import Skeleton from '@/components/shared/skeleton/Skeleton.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {useAnalysis} from '@/context/analysis.ts';
import {useFetch} from '@/hooks/useFetch.ts';
import type {AnomalyList, DashboardSummary, Meter, Page} from '@/interfaces/interfaces.ts';
import {formatNumber} from '@/lib/format.ts';

const DashboardPage = () =>
{
    const {completedVersion} = useAnalysis();
    const summary = useFetch<DashboardSummary>('/dashboard/summary');
    const anomalies = useFetch<AnomalyList>('/anomalies');
    const meters = useFetch<Page<Meter>>('/meters?limit=100&sortBy=code');

    const {execute: refreshSummary} = summary;
    const {execute: refreshAnomalies} = anomalies;
    const {execute: refreshMeters} = meters;

    useEffect(() =>
    {
        if (completedVersion === 0) return;

        void refreshSummary();
        void refreshAnomalies();
        void refreshMeters();
    }, [completedVersion, refreshSummary, refreshAnomalies, refreshMeters]);

    return (
        <div className={styles.page}>
            <PageHeader
                title='Dashboard'
                description='Estado de la flota, último análisis IA y anomalías ordenadas por prioridad de atención.'
            />

            {summary.error && !summary.data
                ? <Card><StateMessage variant='error' title='No se pudo cargar el resumen' description={summary.error.message} onRetry={() => void refreshSummary()}/></Card>
                : (
                    <div className={styles.kpis}>
                        {summary.data ? (
                            <>
                                <KpiCard
                                    label='Medidores'
                                    icon={Gauge}
                                    value={formatNumber(summary.data.metersCount)}
                                    footer={summary.data.metersWithAlerts > 0
                                        ? <Badge tone='explainable' dot>{summary.data.metersWithAlerts} con alerta</Badge>
                                        : <Badge tone='accent' dot>Todos normales</Badge>}
                                />
                                <KpiCard
                                    label='Lecturas'
                                    icon={Activity}
                                    value={formatNumber(summary.data.readingsCount)}
                                    footer={<span className={styles.muted}>Lecturas horarias procesadas</span>}
                                />
                                <KpiCard
                                    label='Consumo total'
                                    icon={Zap}
                                    value={formatNumber(summary.data.totalKwh, 2)}
                                    unit='kWh'
                                    footer={<span className={styles.muted}>Suma de todas las lecturas</span>}
                                />
                                <KpiCard
                                    label='Anomalías abiertas'
                                    icon={AlertTriangle}
                                    tone='danger'
                                    value={formatNumber(summary.data.openAnomalies)}
                                    footer={summary.data.highPriorityAnomalies > 0
                                        ? <Badge tone='real' dot pulse>{summary.data.highPriorityAnomalies} de alta prioridad</Badge>
                                        : <span className={styles.muted}>Sin alta prioridad</span>}
                                />
                            </>
                        ) : Array.from({length: 4}, (_, index) => <Skeleton key={index} height='132px' radius='var(--radius-lg)'/>)}
                    </div>
                )}

            <LastAnalysisCard/>

            <div className={styles.columns}>
                <Card
                    title='Prioridad de atención'
                    subtitle='Ranking del último análisis por prioridad: severidad × magnitud × confianza'
                    className={styles.priority}
                    actions={anomalies.data && <Badge tone='neutral'>{anomalies.data.data.length} casos</Badge>}
                >
                    {anomalies.error && !anomalies.data && (
                        <StateMessage variant='error' title='No se pudieron cargar las anomalías' description={anomalies.error.message} onRetry={() => void refreshAnomalies()}/>
                    )}
                    {!anomalies.data && !anomalies.error && Array.from({length: 4}, (_, index) => <Skeleton key={index} height='72px'/>)}
                    {anomalies.data && anomalies.data.data.length === 0 && (
                        <StateMessage icon={ListChecks} title='Sin anomalías' description='Ejecuta un análisis IA para detectar y priorizar anomalías.'/>
                    )}
                    {anomalies.data && anomalies.data.data.length > 0 && <PriorityList anomalies={anomalies.data.data}/>}
                </Card>

                <div className={styles.side}>
                    <Card title='Variación vs. línea base' subtitle='Consumo de las últimas 48 h frente a los 7 días previos'>
                        {meters.error && !meters.data && (
                            <StateMessage variant='error' title='No se pudieron cargar los medidores' description={meters.error.message} onRetry={() => void refreshMeters()}/>
                        )}
                        {!meters.data && !meters.error && <Skeleton height='240px'/>}
                        {meters.data && <VariationChart meters={meters.data.data}/>}
                    </Card>

                    <Card title='Medidores con alerta' subtitle='Estado asignado por el último análisis'>
                        {!meters.data && !meters.error && <Skeleton height='96px'/>}
                        {meters.data && <AlertMeters meters={meters.data.data}/>}
                    </Card>
                </div>
            </div>
        </div>
    );
}

export default DashboardPage;
