import styles from './statusworkflow.module.css';
import {useState} from 'react';
import {toast} from 'sonner';
import {CheckCircle2, RotateCcw, Search, XCircle, type LucideIcon} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import Button from '@/components/shared/button/Button.tsx';
import type {AnomalyStatus} from '@/interfaces/enums.ts';
import type {AnomalyDetail, UpdateAnomalyStatusRequest} from '@/interfaces/interfaces.ts';
import {ApiError, apiRequest} from '@/lib/api.ts';
import {ALLOWED_TRANSITIONS, ANOMALY_STATUS, TRANSITION_ACTION} from '@/lib/labels.ts';

const MAX_NOTE_LENGTH = 500;

const TRANSITION_ICON: Record<AnomalyStatus, LucideIcon> = {
    OPEN: RotateCcw,
    INVESTIGATING: Search,
    RESOLVED: CheckCircle2,
    DISMISSED: XCircle,
};

interface StatusWorkflowProps {
    anomaly: AnomalyDetail;
    onUpdated: (anomaly: AnomalyDetail) => void;
}

const StatusWorkflow = ({anomaly, onUpdated}: StatusWorkflowProps) =>
{
    const [note, setNote] = useState('');
    const [pending, setPending] = useState<AnomalyStatus | null>(null);

    const status = ANOMALY_STATUS[anomaly.status];
    const transitions = ALLOWED_TRANSITIONS[anomaly.status];
    const tooLong = note.length > MAX_NOTE_LENGTH;

    const changeStatus = async (next: AnomalyStatus) =>
    {
        if (tooLong) return;

        setPending(next);

        try
        {
            const body: UpdateAnomalyStatusRequest = {status: next, note: note.trim() || undefined};
            const updated = await apiRequest<AnomalyDetail>(`/anomalies/${anomaly.id}/status`, {method: 'PATCH', body});

            toast.success(`Anomalía ${ANOMALY_STATUS[next].label.toLowerCase()}`);
            setNote('');
            onUpdated(updated);
        }
        catch (error)
        {
            toast.error('No se pudo cambiar el estado', {description: ApiError.from(error).message});
        }
        finally
        {
            setPending(null);
        }
    }

    return (
        <div className={styles.workflow}>
            <div className={styles.current}>
                <span>Estado actual</span>
                <Badge tone={status.tone} dot>{status.label}</Badge>
            </div>

            <label className={styles.noteField}>
                <span className={styles.noteLabel}>Nota (opcional)</span>
                <textarea
                    value={note}
                    onChange={event => setNote(event.target.value)}
                    placeholder='Qué se revisó, qué se encontró o por qué se descarta…'
                    aria-invalid={tooLong}
                    rows={3}
                />
                <span className={`${styles.counter} tabular ${tooLong ? styles.over : ''}`}>{note.length}/{MAX_NOTE_LENGTH}</span>
            </label>

            <div className={styles.actions}>
                {transitions.map(next =>
                {
                    const Icon = TRANSITION_ICON[next];

                    return (
                        <Button
                            key={next}
                            size='sm'
                            variant={next === 'INVESTIGATING' || next === 'RESOLVED' ? 'primary' : 'secondary'}
                            icon={<Icon size={15}/>}
                            isLoading={pending === next}
                            disabled={pending !== null || tooLong}
                            onClick={() => void changeStatus(next)}
                        >
                            {TRANSITION_ACTION[next]}
                        </Button>
                    );
                })}
            </div>

            {anomaly.resolutionNote && (
                <blockquote className={styles.savedNote}>
                    <span>Última nota registrada</span>
                    <p>{anomaly.resolutionNote}</p>
                </blockquote>
            )}
        </div>
    );
}

export default StatusWorkflow;
