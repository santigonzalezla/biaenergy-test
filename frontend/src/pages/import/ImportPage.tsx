import styles from './import.module.css';
import {useState} from 'react';
import {Info} from 'lucide-react';
import RunAnalysisButton from '@/components/analysis/runanalysisbutton/RunAnalysisButton.tsx';
import ImportCard from '@/components/import/importcard/ImportCard.tsx';
import ImportHistory from '@/components/import/importhistory/ImportHistory.tsx';
import Card from '@/components/shared/card/Card.tsx';
import PageHeader from '@/components/shared/pageheader/PageHeader.tsx';
import Skeleton from '@/components/shared/skeleton/Skeleton.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {useFetch} from '@/hooks/useFetch.ts';
import type {ImportBatchList} from '@/interfaces/interfaces.ts';

const ImportPage = () =>
{
    const [hasNewData, setHasNewData] = useState(false);
    const history = useFetch<ImportBatchList>('/imports?limit=20');
    const refreshHistory = () => void history.execute();

    return (
        <div className={styles.page}>
            <PageHeader
                title='Importar datos'
                description='Carga lecturas horarias y eventos operativos desde archivos CSV. Las filas que ya existen se omiten, así que puedes volver a subir el mismo archivo sin duplicar datos.'
            />

            {hasNewData && (
                <div className={styles.banner} role='status'>
                    <Info size={18}/>
                    <p>Hay datos nuevos. Ejecuta un análisis IA para actualizar las anomalías y el estado de los medidores.</p>
                    <RunAnalysisButton size='sm'/>
                </div>
            )}

            <div className={styles.grid}>
                <ImportCard
                    title='Lecturas'
                    description='Consumo, voltaje, corriente y factor de potencia por hora. Los medidores nuevos se crean automáticamente.'
                    endpoint='/readings/import'
                    columns={['meter_id', 'timestamp', 'consumption_kwh', 'voltage_v', 'current_a', 'power_factor', 'status']}
                    example='M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK'
                    onImported={result => result.inserted > 0 && setHasNewData(true)}
                    onSettled={refreshHistory}
                />
                <ImportCard
                    title='Eventos'
                    description='Eventos operativos que ayudan a explicar cambios de consumo, como cortes programados o cambios de producción.'
                    endpoint='/events/import'
                    columns={['meter_id', 'event_timestamp', 'event_type', 'description']}
                    example='M-106,2026-09-08 00:00,SCHEDULED_OUTAGE,Scheduled maintenance outage for 12 hours'
                    onImported={result => result.inserted > 0 && setHasNewData(true)}
                    onSettled={refreshHistory}
                />
            </div>

            <Card title='Historial de importaciones' subtitle='Últimos 20 archivos subidos, incluidos los rechazados'>
                {history.error && !history.data && (
                    <StateMessage variant='error' title='No se pudo cargar el historial' description={history.error.message} onRetry={refreshHistory}/>
                )}
                {!history.data && !history.error && <Skeleton height='120px'/>}
                {history.data && <ImportHistory batches={history.data.data}/>}
            </Card>

            <p className={styles.note}>
                Las fechas se interpretan en hora de Bogotá. Tipos de evento válidos: OPERATIONAL_CHANGE, SCHEDULED_OUTAGE, DATA_QUALITY, MAINTENANCE y UNKNOWN.
            </p>
        </div>
    );
}

export default ImportPage;
