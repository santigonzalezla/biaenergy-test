import styles from './anomalies.module.css';
import {useEffect} from 'react';
import {useSearchParams} from 'react-router';
import {FilterX, ListChecks} from 'lucide-react';
import AnomalyCard from '@/components/anomalies/anomalycard/AnomalyCard.tsx';
import RunAnalysisButton from '@/components/analysis/runanalysisbutton/RunAnalysisButton.tsx';
import Button from '@/components/shared/button/Button.tsx';
import Card from '@/components/shared/card/Card.tsx';
import FilterPills, {type FilterOption} from '@/components/shared/filterpills/FilterPills.tsx';
import PageHeader from '@/components/shared/pageheader/PageHeader.tsx';
import Skeleton from '@/components/shared/skeleton/Skeleton.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {useAnalysis} from '@/context/analysis.ts';
import {useFetch} from '@/hooks/useFetch.ts';
import {AnomalySeverity, AnomalyStatus, AnomalyType} from '@/interfaces/enums.ts';
import type {Analysis, AnomalyList, AnomalySummary} from '@/interfaces/interfaces.ts';
import {formatDateTime} from '@/lib/format.ts';
import {ANOMALY_STATUS, ANOMALY_TYPE, SEVERITY, type LabelWithTone} from '@/lib/labels.ts';

type FilterKey = 'type' | 'severity' | 'status';

const parse = <T extends string>(value: string | null, allowed: Record<string, T>): T | null =>
{
    return value && (Object.values(allowed) as string[]).includes(value) ? value as T : null;
}

const optionsFor = <T extends string>(labels: Record<T, LabelWithTone>, anomalies: AnomalySummary[], key: FilterKey): FilterOption<T>[] =>
{
    return (Object.keys(labels) as T[]).map(value => ({
        value,
        label: labels[value].label,
        count: anomalies.filter(anomaly => anomaly[key] === value).length,
    }));
}

const AnomaliesPage = () =>
{
    const [params, setParams] = useSearchParams();
    const {completedVersion} = useAnalysis();

    const type = parse(params.get('type'), AnomalyType);
    const severity = parse(params.get('severity'), AnomalySeverity);
    const status = parse(params.get('status'), AnomalyStatus);

    const anomalies = useFetch<AnomalyList>('/anomalies');
    const analysisId = anomalies.data?.analysisId ?? null;
    const analysis = useFetch<Analysis>(analysisId ? `/ai/analysis/${analysisId}` : null);
    const {execute: refresh} = anomalies;

    useEffect(() =>
    {
        if (completedVersion > 0) void refresh();
    }, [completedVersion, refresh]);

    const setFilter = (key: FilterKey, value: string | null) =>
    {
        setParams(previous =>
        {
            const next = new URLSearchParams(previous);

            if (value) next.set(key, value);
            else next.delete(key);

            return next;
        }, {replace: true});
    }

    const all = anomalies.data?.data ?? [];
    const visible = all.filter(anomaly =>
        (!type || anomaly.type === type) && (!severity || anomaly.severity === severity) && (!status || anomaly.status === status));
    const hasFilters = Boolean(type || severity || status);

    const description = analysis.data?.finishedAt
        ? `Resultado del análisis #${analysis.data.numId} · ${formatDateTime(analysis.data.finishedAt)}. Ordenadas por prioridad de atención.`
        : 'Anomalías del último análisis, ordenadas por prioridad de atención.';

    return (
        <div className={styles.page}>
            <PageHeader title='Anomalías IA' description={description}/>

            {anomalies.error && !anomalies.data && (
                <Card>
                    <StateMessage variant='error' title='No se pudieron cargar las anomalías' description={anomalies.error.message} onRetry={() => void refresh()}/>
                </Card>
            )}

            {!anomalies.data && !anomalies.error && (
                <div className={styles.list}>
                    {Array.from({length: 4}, (_, index) => <Skeleton key={index} height='150px' radius='var(--radius-lg)'/>)}
                </div>
            )}

            {anomalies.data && all.length === 0 && (
                <Card>
                    <StateMessage
                        icon={ListChecks}
                        title='Aún no hay anomalías'
                        description='Ejecuta un análisis IA para detectar, clasificar y priorizar anomalías en los medidores.'
                        action={<RunAnalysisButton/>}
                    />
                </Card>
            )}

            {anomalies.data && all.length > 0 && (
                <>
                    <Card className={styles.filters}>
                        <FilterPills label='Tipo' options={optionsFor(ANOMALY_TYPE, all, 'type')} selected={type} onChange={value => setFilter('type', value)}/>
                        <FilterPills label='Severidad' options={optionsFor(SEVERITY, all, 'severity')} selected={severity} onChange={value => setFilter('severity', value)}/>
                        <FilterPills label='Estado' options={optionsFor(ANOMALY_STATUS, all, 'status')} selected={status} onChange={value => setFilter('status', value)}/>
                    </Card>

                    <p className={styles.count} aria-live='polite'>
                        {visible.length === all.length ? `${all.length} anomalías` : `${visible.length} de ${all.length} anomalías`}
                    </p>

                    {visible.length === 0
                        ? (
                            <Card>
                                <StateMessage
                                    icon={FilterX}
                                    title='Ninguna anomalía coincide con los filtros'
                                    action={hasFilters && <Button size='sm' onClick={() => setParams({}, {replace: true})}>Limpiar filtros</Button>}
                                />
                            </Card>
                        )
                        : (
                            <div className={styles.list}>
                                {visible.map(anomaly => (
                                    <AnomalyCard key={anomaly.id} anomaly={anomaly} rank={all.indexOf(anomaly) + 1}/>
                                ))}
                            </div>
                        )}
                </>
            )}
        </div>
    );
}

export default AnomaliesPage;
