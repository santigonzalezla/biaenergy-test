import styles from './consumptionchart.module.css';
import {Area, CartesianGrid, ComposedChart, ReferenceArea, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis, type TooltipContentProps, type TooltipValueType} from 'recharts';
import type {Reading} from '@/interfaces/interfaces.ts';
import {formatDateTime, formatDayMonth, formatNumber} from '@/lib/format.ts';

interface ConsumptionChartProps {
    readings: Reading[];
    baselineKwh: number | null;
    changeAt: string | null;
}

interface Point {
    time: number;
    consumption: number;
}

const ChartTooltip = ({active, payload}: TooltipContentProps<TooltipValueType, number | string>) =>
{
    if (!active || !payload?.length) return null;

    const point = payload[0].payload as Point;

    return (
        <div className={styles.tooltip}>
            <span>{formatDateTime(new Date(point.time).toISOString())}</span>
            <strong className='tabular'>{formatNumber(point.consumption, 2)} kWh</strong>
        </div>
    );
}

const ConsumptionChart = ({readings, baselineKwh, changeAt}: ConsumptionChartProps) =>
{
    const data: Point[] = readings.map(reading => ({time: Date.parse(reading.timestamp), consumption: reading.consumptionKwh}));
    const changeTime = changeAt ? Date.parse(changeAt) : null;
    const lastTime = data.at(-1)?.time;
    const showChange = changeTime !== null && lastTime !== undefined && changeTime <= lastTime;

    return (
        <div className={styles.chart} role='img' aria-label='Consumo horario del medidor con su línea base'>
            <ResponsiveContainer width='100%' height={300}>
                <ComposedChart data={data} margin={{top: 24, right: 12, bottom: 0, left: -8}}>
                    <defs>
                        <linearGradient id='consumption-fill' x1='0' y1='0' x2='0' y2='1'>
                            <stop offset='0%' stopColor='var(--chart-consumption)' stopOpacity={0.35}/>
                            <stop offset='100%' stopColor='var(--chart-consumption)' stopOpacity={0}/>
                        </linearGradient>
                    </defs>
                    <CartesianGrid vertical={false} stroke='var(--chart-grid)' strokeDasharray='3 3'/>
                    <XAxis
                        dataKey='time'
                        type='number'
                        scale='time'
                        domain={['dataMin', 'dataMax']}
                        tickFormatter={formatDayMonth}
                        tickLine={false}
                        axisLine={false}
                        minTickGap={32}
                        tick={{fill: 'var(--text-muted)', fontSize: 11}}
                    />
                    <YAxis tickLine={false} axisLine={false} width={48} tick={{fill: 'var(--text-muted)', fontSize: 11}} unit=' kWh'/>
                    {showChange && (
                        <ReferenceArea x1={changeTime} x2={lastTime} fill='var(--chart-threshold)' strokeOpacity={0}/>
                    )}
                    {baselineKwh !== null && (
                        <ReferenceLine
                            y={baselineKwh}
                            stroke='var(--chart-baseline)'
                            strokeDasharray='6 4'
                            strokeWidth={1.5}
                            label={{value: `Línea base ${formatNumber(baselineKwh, 1)} kWh/h`, position: 'insideBottomLeft', fill: 'var(--text-muted)', fontSize: 11}}
                        />
                    )}
                    {showChange && (
                        <ReferenceLine
                            x={changeTime}
                            stroke='var(--anomaly-real)'
                            strokeDasharray='4 3'
                            strokeWidth={2}
                            label={{value: 'Punto de cambio', position: 'top', fill: 'var(--anomaly-real-text)', fontSize: 11, fontWeight: 700}}
                        />
                    )}
                    <Tooltip content={ChartTooltip}/>
                    <Area
                        type='monotone'
                        dataKey='consumption'
                        stroke='var(--chart-consumption)'
                        strokeWidth={2}
                        fill='url(#consumption-fill)'
                        isAnimationActive={false}
                    />
                </ComposedChart>
            </ResponsiveContainer>

            <ul className={styles.legend}>
                <li><span className={styles.line}/>Consumo horario</li>
                {baselineKwh !== null && <li><span className={styles.dashed}/>Línea base</li>}
                {showChange && <li><span className={styles.band}/>Periodo anómalo</li>}
            </ul>
        </div>
    );
}

export default ConsumptionChart;
