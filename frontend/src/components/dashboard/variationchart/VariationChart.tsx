import styles from './variationchart.module.css';
import {Bar, BarChart, CartesianGrid, Cell, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis, type TooltipContentProps, type TooltipValueType} from 'recharts';
import {MeterStatus} from '@/interfaces/enums.ts';
import type {Meter} from '@/interfaces/interfaces.ts';
import {formatKwh, formatPercent} from '@/lib/format.ts';
import {METER_STATUS} from '@/lib/labels.ts';

interface VariationChartProps {
    meters: Meter[];
}

interface ChartPoint {
    code: string;
    status: MeterStatus;
    variationPct: number;
    baselineKwh: number;
    recentKwh: number;
}

const BAR_COLOR: Record<MeterStatus, string> = {
    OK: 'var(--chart-consumption)',
    ALERT: 'var(--anomaly-explainable)',
    CRITICAL: 'var(--anomaly-real)',
};

const ChartTooltip = ({active, payload}: TooltipContentProps<TooltipValueType, number | string>) =>
{
    if (!active || !payload?.length) return null;

    const point = payload[0].payload as ChartPoint;

    return (
        <div className={styles.tooltip}>
            <strong className='mono'>{point.code}</strong>
            <span>{METER_STATUS[point.status].label}</span>
            <span>Línea base: <b className='tabular'>{formatKwh(point.baselineKwh)}/h</b></span>
            <span>Reciente (48 h): <b className='tabular'>{formatKwh(point.recentKwh)}/h</b></span>
            <span>Variación: <b className='tabular'>{formatPercent(point.variationPct)}</b></span>
        </div>
    );
}

const VariationChart = ({meters}: VariationChartProps) =>
{
    const data: ChartPoint[] = meters
        .filter(meter => meter.stats !== null)
        .map(meter => ({
            code: meter.code,
            status: meter.status,
            variationPct: meter.stats!.variationPct,
            baselineKwh: meter.stats!.baselineKwh,
            recentKwh: meter.stats!.recentKwh,
        }));

    return (
        <div className={styles.chart} role='img' aria-label='Variación del consumo reciente frente a la línea base por medidor'>
            <ResponsiveContainer width='100%' height={240}>
                <BarChart data={data} margin={{top: 8, right: 8, bottom: 0, left: -12}}>
                    <CartesianGrid vertical={false} stroke='var(--chart-grid)' strokeDasharray='3 3'/>
                    <XAxis dataKey='code' tickLine={false} axisLine={false} tick={{fill: 'var(--text-muted)', fontSize: 11}} tickFormatter={code => code.replace('M-', '')}/>
                    <YAxis tickLine={false} axisLine={false} tick={{fill: 'var(--text-muted)', fontSize: 11}} tickFormatter={value => `${value}%`}/>
                    <ReferenceLine y={0} stroke='var(--border-strong)'/>
                    <Tooltip content={ChartTooltip} cursor={{fill: 'var(--bg-hover)'}}/>
                    <Bar dataKey='variationPct' radius={[6, 6, 0, 0]} maxBarSize={36}>
                        {data.map(point => <Cell key={point.code} fill={BAR_COLOR[point.status]}/>)}
                    </Bar>
                </BarChart>
            </ResponsiveContainer>

            <ul className={styles.legend}>
                {Object.values(MeterStatus).map(status => (
                    <li key={status}>
                        <span className={styles.swatch} style={{background: BAR_COLOR[status]}}/>
                        {METER_STATUS[status].label}
                    </li>
                ))}
            </ul>
        </div>
    );
}

export default VariationChart;
