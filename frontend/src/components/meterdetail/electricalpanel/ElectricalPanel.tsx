import styles from './electricalpanel.module.css';
import {useState} from 'react';
import {CartesianGrid, Line, LineChart, ReferenceArea, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import type {Reading} from '@/interfaces/interfaces.ts';
import {formatDayMonth, formatNumber, formatPercent} from '@/lib/format.ts';
import {average, changePct, extreme, splitAt, type ReadingField} from '@/lib/series.ts';

type Variable = Extract<ReadingField, 'voltage' | 'current' | 'powerFactor'>;

interface ElectricalPanelProps {
    readings: Reading[];
    nominalVoltage: number;
    changeAt: string | null;
}

interface Threshold {
    kind: 'band' | 'min' | 'max';
    low?: number;
    high?: number;
    value?: number;
    label: string;
}

interface VariableConfig {
    label: string;
    unit: string;
    digits: number;
    color: string;
    extremeLabel: string;
    extremePick: 'min' | 'max';
}

const VARIABLES: Record<Variable, VariableConfig> = {
    voltage: {label: 'Voltaje', unit: 'V', digits: 1, color: 'var(--chart-voltage)', extremeLabel: 'Mínimo', extremePick: 'min'},
    current: {label: 'Corriente', unit: 'A', digits: 1, color: 'var(--chart-current)', extremeLabel: 'Pico', extremePick: 'max'},
    powerFactor: {label: 'Factor de potencia', unit: '', digits: 2, color: 'var(--chart-power-factor)', extremeLabel: 'Mínimo', extremePick: 'min'},
};

const POWER_FACTOR_MIN = 0.8;
const OVERCURRENT_FACTOR = 1.5;
const VOLTAGE_TOLERANCE = 0.05;

const thresholdFor = (variable: Variable, nominalVoltage: number, before: Reading[]): Threshold | null =>
{
    if (variable === 'voltage')
    {
        const low = nominalVoltage * (1 - VOLTAGE_TOLERANCE);
        const high = nominalVoltage * (1 + VOLTAGE_TOLERANCE);

        return {kind: 'band', low, high, label: `Banda ±5 %: ${formatNumber(low, 0)}–${formatNumber(high, 0)} V`};
    }

    if (variable === 'powerFactor') return {kind: 'min', value: POWER_FACTOR_MIN, label: 'Mínimo aceptable: 0,80'};

    const previousPeak = extreme(before, 'current', 'max');

    if (previousPeak === null) return null;

    const value = previousPeak * OVERCURRENT_FACTOR;

    return {kind: 'max', value, label: `Sobrecorriente: > 1,5 × pico previo (${formatNumber(value, 0)} A)`};
}

const ElectricalPanel = ({readings, nominalVoltage, changeAt}: ElectricalPanelProps) =>
{
    const [variable, setVariable] = useState<Variable>('powerFactor');
    const config = VARIABLES[variable];
    const {before, after} = splitAt(readings, changeAt);
    const hasChange = after.length > 0 && before.length > 0;
    const threshold = thresholdFor(variable, nominalVoltage, before);

    const beforeAvg = average(before, variable);
    const afterAvg = average(after, variable);
    const worst = extreme(hasChange ? after : readings, variable, config.extremePick);
    const format = (value: number | null) => value === null ? '—' : `${formatNumber(value, config.digits)}${config.unit ? ` ${config.unit}` : ''}`;

    const data = readings.map(reading => ({time: Date.parse(reading.timestamp), value: reading[variable]}));

    return (
        <div className={styles.panel}>
            <div className={styles.tabs} role='tablist' aria-label='Variable eléctrica'>
                {(Object.keys(VARIABLES) as Variable[]).map(key => (
                    <button
                        key={key}
                        type='button'
                        role='tab'
                        aria-selected={variable === key}
                        className={`${styles.tab} ${variable === key ? styles.active : ''}`}
                        onClick={() => setVariable(key)}
                    >
                        <span className={styles.swatch} style={{background: VARIABLES[key].color}}/>
                        {VARIABLES[key].label}
                    </button>
                ))}
            </div>

            <div className={styles.stats} role='tabpanel' aria-label={config.label}>
                <div className={styles.stat}>
                    <span>{hasChange ? 'Promedio antes del cambio' : 'Promedio'}</span>
                    <strong className='tabular'>{format(beforeAvg)}</strong>
                </div>
                {hasChange && (
                    <div className={`${styles.stat} ${styles.highlight}`}>
                        <span>Promedio después del cambio</span>
                        <strong className='tabular'>{format(afterAvg)}</strong>
                        {changePct(beforeAvg, afterAvg) !== null && (
                            <em className='tabular'>{formatPercent(changePct(beforeAvg, afterAvg)!)}</em>
                        )}
                    </div>
                )}
                <div className={styles.stat}>
                    <span>{config.extremeLabel}{hasChange ? ' después del cambio' : ''}</span>
                    <strong className='tabular'>{format(worst)}</strong>
                    {threshold && <em className={styles.threshold}>{threshold.label}</em>}
                </div>
            </div>

            <ResponsiveContainer width='100%' height={200}>
                <LineChart data={data} margin={{top: 8, right: 12, bottom: 0, left: -8}}>
                    <CartesianGrid vertical={false} stroke='var(--chart-grid)' strokeDasharray='3 3'/>
                    <XAxis dataKey='time' type='number' scale='time' domain={['dataMin', 'dataMax']} tickFormatter={formatDayMonth} tickLine={false} axisLine={false} minTickGap={32} tick={{fill: 'var(--text-muted)', fontSize: 11}}/>
                    <YAxis domain={['auto', 'auto']} tickLine={false} axisLine={false} width={48} tick={{fill: 'var(--text-muted)', fontSize: 11}}/>
                    {threshold?.kind === 'band' && <ReferenceArea y1={threshold.low} y2={threshold.high} fill='var(--accent-soft)' strokeOpacity={0}/>}
                    {threshold?.kind === 'min' && <ReferenceLine y={threshold.value} stroke='var(--anomaly-real)' strokeDasharray='4 3'/>}
                    {threshold?.kind === 'max' && <ReferenceLine y={threshold.value} stroke='var(--anomaly-real)' strokeDasharray='4 3'/>}
                    {hasChange && changeAt && <ReferenceLine x={Date.parse(changeAt)} stroke='var(--anomaly-real)' strokeDasharray='4 3'/>}
                    <Tooltip
                        labelFormatter={value => formatDayMonth(Number(value))}
                        formatter={value => [format(Number(value)), config.label]}
                        contentStyle={{background: 'var(--bg-surface)', border: '1px solid var(--border-primary)', borderRadius: 8, fontSize: 12}}
                    />
                    <Line type='monotone' dataKey='value' stroke={config.color} strokeWidth={2} dot={false} isAnimationActive={false}/>
                </LineChart>
            </ResponsiveContainer>
        </div>
    );
}

export default ElectricalPanel;
