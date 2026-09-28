import {Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import type {HourlyProfile} from '@/interfaces/interfaces.ts';
import {formatNumber} from '@/lib/format.ts';

interface HourlyProfileChartProps {
    before: HourlyProfile;
    after: HourlyProfile | null;
    beforeLabel: string;
    afterLabel: string;
}

const HourlyProfileChart = ({before, after, beforeLabel, afterLabel}: HourlyProfileChartProps) =>
{
    const afterByHour = new Map(after?.hours.map(value => [value.hour, value.avgKwh]) ?? []);
    const data = Array.from({length: 24}, (_, hour) => ({
        hour: `${String(hour).padStart(2, '0')}h`,
        before: before.hours.find(value => value.hour === hour)?.avgKwh ?? null,
        after: afterByHour.get(hour) ?? null,
    }));

    return (
        <div role='img' aria-label='Perfil horario promedio antes y después del cambio'>
            <ResponsiveContainer width='100%' height={240}>
                <BarChart data={data} margin={{top: 8, right: 8, bottom: 0, left: -8}} barGap={2}>
                    <CartesianGrid vertical={false} stroke='var(--chart-grid)' strokeDasharray='3 3'/>
                    <XAxis dataKey='hour' tickLine={false} axisLine={false} interval={2} tick={{fill: 'var(--text-muted)', fontSize: 11}}/>
                    <YAxis tickLine={false} axisLine={false} width={40} tick={{fill: 'var(--text-muted)', fontSize: 11}}/>
                    <Tooltip
                        cursor={{fill: 'var(--bg-hover)'}}
                        formatter={value => `${formatNumber(Number(value), 2)} kWh`}
                        contentStyle={{background: 'var(--bg-surface)', border: '1px solid var(--border-primary)', borderRadius: 8, fontSize: 12}}
                    />
                    <Legend wrapperStyle={{fontSize: 12}}/>
                    <Bar dataKey='before' name={beforeLabel} fill='var(--chart-baseline)' radius={[3, 3, 0, 0]} isAnimationActive={false}/>
                    {after && <Bar dataKey='after' name={afterLabel} fill='var(--anomaly-real)' radius={[3, 3, 0, 0]} isAnimationActive={false}/>}
                </BarChart>
            </ResponsiveContainer>
        </div>
    );
}

export default HourlyProfileChart;
