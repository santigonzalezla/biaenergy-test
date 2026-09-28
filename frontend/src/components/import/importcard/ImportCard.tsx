import styles from './importcard.module.css';
import {useState} from 'react';
import {toast} from 'sonner';
import {AlertTriangle, CheckCircle2, Info, Upload} from 'lucide-react';
import Dropzone from '@/components/import/dropzone/Dropzone.tsx';
import Button from '@/components/shared/button/Button.tsx';
import Card from '@/components/shared/card/Card.tsx';
import type {ImportResult, RowError} from '@/interfaces/interfaces.ts';
import {ApiError, apiRequest} from '@/lib/api.ts';
import {formatDateTime, formatNumber} from '@/lib/format.ts';

interface ImportCardProps {
    title: string;
    description: string;
    endpoint: string;
    columns: string[];
    example: string;
    onImported?: (result: ImportResult) => void;
    onSettled?: () => void;
}

type Outcome =
    | {kind: 'success'; result: ImportResult}
    | {kind: 'rows'; errors: RowError[]}
    | {kind: 'error'; message: string};

const ImportCard = ({title, description, endpoint, columns, example, onImported, onSettled}: ImportCardProps) =>
{
    const [file, setFile] = useState<File | null>(null);
    const [isUploading, setIsUploading] = useState(false);
    const [outcome, setOutcome] = useState<Outcome | null>(null);

    const selectFile = (next: File | null) =>
    {
        setFile(next);
        setOutcome(null);
    }

    const upload = async () =>
    {
        if (!file) return;

        const form = new FormData();
        form.append('file', file);
        setIsUploading(true);

        try
        {
            const result = await apiRequest<ImportResult>(endpoint, {method: 'POST', body: form});

            setOutcome({kind: 'success', result});
            setFile(null);
            toast.success(`${title}: ${formatNumber(result.inserted)} filas nuevas`);
            onImported?.(result);
        }
        catch (caught)
        {
            const error = ApiError.from(caught);

            if (error.code === 'VALIDATION_ERROR' && Array.isArray(error.details)) setOutcome({kind: 'rows', errors: error.details});
            else if (error.code === 'INVALID_CSV') setOutcome({kind: 'error', message: `El archivo no tiene el formato esperado: ${error.message}`});
            else setOutcome({kind: 'error', message: error.message});
        }
        finally
        {
            setIsUploading(false);
            onSettled?.();
        }
    }

    return (
        <Card title={title} subtitle={description}>
            <div className={styles.format}>
                <span className={styles.formatLabel}>Columnas esperadas</span>
                <div className={styles.columns}>
                    {columns.map(column => <code key={column}>{column}</code>)}
                </div>
                <span className={styles.formatLabel}>Ejemplo</span>
                <code className={styles.example}>{example}</code>
            </div>

            <Dropzone
                label={`Archivo CSV de ${title.toLowerCase()}`}
                file={file}
                disabled={isUploading}
                onFileChange={selectFile}
                onReject={message => setOutcome({kind: 'error', message})}
            />

            <Button variant='primary' icon={<Upload size={16}/>} onClick={() => void upload()} disabled={!file} isLoading={isUploading}>
                {isUploading ? 'Importando…' : 'Importar archivo'}
            </Button>

            {outcome?.kind === 'success' && (
                <div className={styles.success} role='status'>
                    <div className={styles.outcomeTitle}>
                        <CheckCircle2 size={18}/>
                        Importación completada
                    </div>
                    <dl className={styles.stats}>
                        <div><dt>Filas leídas</dt><dd className='tabular'>{formatNumber(outcome.result.rows)}</dd></div>
                        <div><dt>Insertadas</dt><dd className='tabular'>{formatNumber(outcome.result.inserted)}</dd></div>
                        <div><dt>Omitidas (ya existían)</dt><dd className='tabular'>{formatNumber(outcome.result.skipped)}</dd></div>
                        <div><dt>Medidores creados</dt><dd className='tabular'>{formatNumber(outcome.result.metersCreated)}</dd></div>
                        {outcome.result.metersRestored > 0 && (
                            <div><dt>Medidores restaurados</dt><dd className='tabular'>{formatNumber(outcome.result.metersRestored)}</dd></div>
                        )}
                    </dl>
                    {outcome.result.previouslyImportedAt && (
                        <p className={styles.repeated}>
                            <Info size={14}/>
                            Este mismo archivo ya se había importado el {formatDateTime(outcome.result.previouslyImportedAt)}. Las filas repetidas se omitieron.
                        </p>
                    )}
                </div>
            )}

            {outcome?.kind === 'rows' && (
                <div className={styles.failure} role='alert'>
                    <div className={styles.outcomeTitle}>
                        <AlertTriangle size={18}/>
                        No se importó nada: hay filas con errores
                    </div>
                    <p className={styles.failureHint}>
                        Corrige estas filas y vuelve a subir el archivo. Se muestran como máximo las primeras 20.
                    </p>
                    <table className={styles.errors}>
                        <thead>
                            <tr>
                                <th scope='col'>Fila</th>
                                <th scope='col'>Columna</th>
                                <th scope='col'>Error</th>
                            </tr>
                        </thead>
                        <tbody>
                            {outcome.errors.map(error => (
                                <tr key={`${error.line}-${error.column ?? ''}`}>
                                    <td className='tabular'>{error.line}</td>
                                    <td><code>{error.column ?? '—'}</code></td>
                                    <td>{error.message}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {outcome?.kind === 'error' && (
                <div className={styles.failure} role='alert'>
                    <div className={styles.outcomeTitle}>
                        <AlertTriangle size={18}/>
                        {outcome.message}
                    </div>
                </div>
            )}
        </Card>
    );
}

export default ImportCard;
