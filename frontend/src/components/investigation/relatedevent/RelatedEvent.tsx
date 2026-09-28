import styles from './relatedevent.module.css';
import {CalendarClock, CalendarX} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import type {EventType} from '@/interfaces/enums.ts';
import type {EventReference} from '@/interfaces/interfaces.ts';
import {formatDateTime} from '@/lib/format.ts';
import {EVENT_TYPE, type Tone} from '@/lib/labels.ts';

interface RelatedEventProps {
    event: EventReference | null;
}

const VERDICT: Record<EventType, {text: string; tone: Tone}> = {
    OPERATIONAL_CHANGE: {text: 'Explica el cambio: corresponde a un cambio operativo registrado.', tone: 'accent'},
    SCHEDULED_OUTAGE: {text: 'Explica la caída: corresponde a un corte programado.', tone: 'accent'},
    MAINTENANCE: {text: 'Puede explicar el cambio: hay un mantenimiento registrado.', tone: 'explainable'},
    DATA_QUALITY: {text: 'Señala un problema de calidad de datos del medidor.', tone: 'dataQuality'},
    UNKNOWN: {text: 'No explica el cambio: no hay un evento operativo que lo justifique.', tone: 'real'},
};

const RelatedEvent = ({event}: RelatedEventProps) =>
{
    if (!event)
    {
        return (
            <StateMessage
                icon={CalendarX}
                title='Sin evento relacionado'
                description='No se registró ningún evento operativo en la ventana de ±6 horas del cambio.'
            />
        );
    }

    const verdict = VERDICT[event.type];

    return (
        <div className={styles.event}>
            <div className={styles.header}>
                <CalendarClock size={18}/>
                <strong>{EVENT_TYPE[event.type]}</strong>
                <time className={styles.time} dateTime={event.timestamp}>{formatDateTime(event.timestamp)}</time>
            </div>
            <p className={styles.description}>{event.description}</p>
            <Badge tone={verdict.tone}>{verdict.text}</Badge>
        </div>
    );
}

export default RelatedEvent;
