import styles from './meterstable.module.css';
import {useNavigate} from 'react-router';
import {ArrowDown, ArrowUp, ArrowUpDown, Pencil, Trash2} from 'lucide-react';
import Badge from '@/components/shared/badge/Badge.tsx';
import {MeterSortField, MeterStatus, SortDirection} from '@/interfaces/enums.ts';
import type {Meter} from '@/interfaces/interfaces.ts';
import {formatNumber, formatPercent, formatShortDateTime} from '@/lib/format.ts';
import {METER_STATUS} from '@/lib/labels.ts';

interface MetersTableProps {
    meters: Meter[];
    sortBy: MeterSortField;
    sortDir: SortDirection;
    onSort: (field: MeterSortField) => void;
    onEdit: (meter: Meter) => void;
    onDelete: (meter: Meter) => void;
}

interface SortHeaderProps {
    field: MeterSortField;
    label: string;
    sortBy: MeterSortField;
    sortDir: SortDirection;
    onSort: (field: MeterSortField) => void;
}

const SortHeader = ({field, label, sortBy, sortDir, onSort}: SortHeaderProps) =>
{
    const isActive = sortBy === field;
    const Icon = !isActive ? ArrowUpDown : sortDir === SortDirection.ASC ? ArrowUp : ArrowDown;

    return (
        <th scope='col' aria-sort={isActive ? (sortDir === SortDirection.ASC ? 'ascending' : 'descending') : 'none'}>
            <button type='button' className={`${styles.sort} ${isActive ? styles.sortActive : ''}`} onClick={() => onSort(field)}>
                {label}
                <Icon size={13}/>
            </button>
        </th>
    );
}

const MetersTable = ({meters, sortBy, sortDir, onSort, onEdit, onDelete}: MetersTableProps) =>
{
    const navigate = useNavigate();
    const sortProps = {sortBy, sortDir, onSort};

    return (
        <div className={styles.wrapper}>
            <table className={styles.table}>
                <thead>
                    <tr>
                        <SortHeader field={MeterSortField.CODE} label='Código' {...sortProps}/>
                        <SortHeader field={MeterSortField.NAME} label='Nombre' {...sortProps}/>
                        <th scope='col' className={styles.hideMd}>Sector</th>
                        <th scope='col' className={`${styles.numeric} ${styles.hideLg}`}>Voltaje</th>
                        <th scope='col' className={`${styles.numeric} ${styles.hideLg}`}>Potencia</th>
                        <SortHeader field={MeterSortField.STATUS} label='Estado' {...sortProps}/>
                        <th scope='col' className={styles.numeric}>Consumo reciente</th>
                        <th scope='col' className={styles.hideMd}>Última lectura</th>
                        <th scope='col'><span className={styles.srOnly}>Acciones</span></th>
                    </tr>
                </thead>
                <tbody>
                    {meters.map(meter =>
                    {
                        const status = METER_STATUS[meter.status];
                        const variation = meter.stats?.variationPct ?? null;
                        const highlight = meter.status === MeterStatus.OK ? '' : styles[status.tone];

                        return (
                            <tr key={meter.id} className={`${styles.row} ${highlight}`} onClick={() => navigate(`/meters/${meter.id}`)}>
                                <td><span className={`${styles.code} mono`}>{meter.code}</span></td>
                                <td>
                                    <div className={styles.name}>
                                        <span>{meter.name}</span>
                                        {meter.location && <span className={styles.location}>{meter.location}</span>}
                                    </div>
                                </td>
                                <td className={styles.hideMd}>{meter.sector || '—'}</td>
                                <td className={`${styles.numeric} ${styles.hideLg} tabular`}>{formatNumber(meter.nominalVoltage)} V</td>
                                <td className={`${styles.numeric} ${styles.hideLg} tabular`}>
                                    {meter.contractedPowerKw === null ? '—' : `${formatNumber(meter.contractedPowerKw)} kW`}
                                </td>
                                <td><Badge tone={status.tone} dot pulse={meter.status === MeterStatus.CRITICAL}>{status.label}</Badge></td>
                                <td className={`${styles.numeric} tabular`}>
                                    {meter.stats ? (
                                        <div className={styles.consumption}>
                                            <span>{formatNumber(meter.stats.recentKwh, 2)} kWh/h</span>
                                            {variation !== null && (
                                                <span className={Math.abs(variation) >= 20 ? styles.variationHigh : styles.variation}>
                                                    {formatPercent(variation)}
                                                </span>
                                            )}
                                        </div>
                                    ) : '—'}
                                </td>
                                <td className={`${styles.hideMd} ${styles.muted}`}>
                                    {meter.stats ? formatShortDateTime(meter.stats.lastReadingAt) : 'Sin lecturas'}
                                </td>
                                <td>
                                    <div className={styles.actions} onClick={event => event.stopPropagation()}>
                                        <button type='button' className={styles.action} onClick={() => onEdit(meter)} aria-label={`Editar ${meter.code}`}>
                                            <Pencil size={15}/>
                                        </button>
                                        <button type='button' className={`${styles.action} ${styles.danger}`} onClick={() => onDelete(meter)} aria-label={`Eliminar ${meter.code}`}>
                                            <Trash2 size={15}/>
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
}

export default MetersTable;
