import styles from './meterstoolbar.module.css';
import {Search, X} from 'lucide-react';
import {MeterStatus} from '@/interfaces/enums.ts';
import {METER_STATUS} from '@/lib/labels.ts';

interface MetersToolbarProps {
    search: string;
    status: MeterStatus | null;
    onSearchChange: (search: string) => void;
    onStatusChange: (status: MeterStatus | null) => void;
}

const STATUS_OPTIONS: (MeterStatus | null)[] = [null, MeterStatus.OK, MeterStatus.ALERT, MeterStatus.CRITICAL];

const MetersToolbar = ({search, status, onSearchChange, onStatusChange}: MetersToolbarProps) =>
{
    return (
        <div className={styles.toolbar}>
            <label className={styles.search}>
                <Search size={16} className={styles.searchIcon}/>
                <input
                    type='search'
                    value={search}
                    onChange={event => onSearchChange(event.target.value)}
                    placeholder='Buscar por código, nombre o ubicación'
                    aria-label='Buscar medidores'
                />
                {search && (
                    <button type='button' className={styles.clear} onClick={() => onSearchChange('')} aria-label='Limpiar búsqueda'>
                        <X size={14}/>
                    </button>
                )}
            </label>

            <div className={styles.pills} role='group' aria-label='Filtrar por estado'>
                {STATUS_OPTIONS.map(option => (
                    <button
                        key={option ?? 'ALL'}
                        type='button'
                        className={`${styles.pill} ${status === option ? styles.active : ''}`}
                        aria-pressed={status === option}
                        onClick={() => onStatusChange(option)}
                    >
                        {option ? METER_STATUS[option].label : 'Todos'}
                    </button>
                ))}
            </div>
        </div>
    );
}

export default MetersToolbar;
