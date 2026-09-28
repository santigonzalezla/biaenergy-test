import styles from './importhistory.module.css';
import {FileClock} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {ImportKind, ImportStatus} from '@/interfaces/enums.ts';
import type {ImportBatch} from '@/interfaces/interfaces.ts';
import {formatNumber, formatShortDateTime} from '@/lib/format.ts';

interface ImportHistoryProps {
    batches: ImportBatch[];
}

const KIND_LABEL: Record<ImportKind, string> = {
    READINGS: 'Lecturas',
    EVENTS: 'Eventos',
};

const ImportHistory = ({batches}: ImportHistoryProps) =>
{
    if (batches.length === 0)
    {
        return <StateMessage icon={FileClock} title='Aún no hay importaciones registradas' description='Cada archivo que subas quedará en este historial, incluidos los que se rechacen.'/>;
    }

    return (
        <div className={styles.wrapper}>
            <table className={styles.table}>
                <thead>
                    <tr>
                        <th scope='col'>#</th>
                        <th scope='col'>Archivo</th>
                        <th scope='col'>Tipo</th>
                        <th scope='col'>Estado</th>
                        <th scope='col'>Resultado</th>
                        <th scope='col' className={styles.hideMd}>Usuario</th>
                        <th scope='col'>Fecha</th>
                    </tr>
                </thead>
                <tbody>
                    {batches.map(batch =>
                    {
                        const failed = batch.status === ImportStatus.FAILED;

                        return (
                            <tr key={batch.id}>
                                <td className={`${styles.muted} tabular`}>{batch.numId}</td>
                                <td>
                                    <div className={styles.file}>
                                        <span className={styles.fileName}>{batch.fileName}</span>
                                        <span className={`${styles.muted} mono`} title={`SHA-256 ${batch.checksum}`}>
                                            {formatNumber(batch.fileSize / 1024, 1)} KB · {batch.checksum.slice(0, 8)}
                                        </span>
                                    </div>
                                </td>
                                <td>{KIND_LABEL[batch.kind]}</td>
                                <td>
                                    <Badge tone={failed ? 'real' : 'accent'} dot>{failed ? 'Rechazado' : 'Completado'}</Badge>
                                </td>
                                <td className={styles.result}>
                                    {failed
                                        ? <span className={styles.error}>{batch.error ?? batch.errorCode}</span>
                                        : (
                                            <span className='tabular'>
                                                {formatNumber(batch.inserted)} nuevas · {formatNumber(batch.skipped)} omitidas
                                                {batch.metersCreated > 0 && ` · ${batch.metersCreated} medidores creados`}
                                                {batch.metersRestored > 0 && ` · ${batch.metersRestored} restaurados`}
                                            </span>
                                        )}
                                </td>
                                <td className={styles.hideMd}>{batch.userName ?? 'Carga inicial'}</td>
                                <td className={styles.muted}>{formatShortDateTime(batch.createdAt)}</td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
}

export default ImportHistory;
