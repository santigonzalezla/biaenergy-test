import styles from './eventtimeline.module.css';
import {CalendarX} from 'lucide-react';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {EventType} from '@/interfaces/enums.ts';
import type {MeterEvent} from '@/interfaces/interfaces.ts';
import {formatDateTime} from '@/lib/format.ts';
import {EVENT_TYPE} from '@/lib/labels.ts';

const CORRELATION_WINDOW_MS = 6 * 60 * 60 * 1000;

interface EventTimelineProps {
    events: MeterEvent[];
    changeAt: string | null;
}

const EventTimeline = ({events, changeAt}: EventTimelineProps) =>
{
    if (events.length === 0)
    {
        return <StateMessage icon={CalendarX} title='Sin eventos en el periodo' description='No hay eventos operativos registrados para este rango.'/>;
    }

    const changeTime = changeAt ? Date.parse(changeAt) : null;
    const sorted = [...events].sort((a, b) => Date.parse(b.timestamp) - Date.parse(a.timestamp));

    return (
        <ol className={styles.timeline}>
            {sorted.map(event =>
            {
                const correlated = changeTime !== null && Math.abs(Date.parse(event.timestamp) - changeTime) <= CORRELATION_WINDOW_MS;
                const unexplained = event.type === EventType.UNKNOWN;

                return (
                    <li key={event.id} className={`${styles.event} ${correlated ? styles.correlated : ''}`}>
                        <span className={`${styles.dot} ${unexplained ? styles.unknown : ''}`}/>
                        <div className={styles.header}>
                            <span className={styles.type}>{EVENT_TYPE[event.type]}</span>
                            <time className={styles.time} dateTime={event.timestamp}>{formatDateTime(event.timestamp)}</time>
                        </div>
                        <p className={styles.description}>{event.description}</p>
                        {correlated && (
                            <span className={styles.note}>
                                {unexplained ? 'Coincide con el cambio, pero no lo explica' : 'Coincide con el punto de cambio'}
                            </span>
                        )}
                    </li>
                );
            })}
        </ol>
    );
}

export default EventTimeline;
