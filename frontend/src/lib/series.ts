import type {IsoDateTime, Reading} from '@/interfaces/interfaces.ts';

const HOUR_MS = 60 * 60 * 1000;
const DAY_MS = 24 * HOUR_MS;

export interface TimeRange {
    from: IsoDateTime;
    to: IsoDateTime;
}

export type ReadingField = 'consumptionKwh' | 'voltage' | 'current' | 'powerFactor';

export interface ReadingWindows {
    before: Reading[];
    after: Reading[];
}

export const rangeEndingAt = (lastReadingAt: IsoDateTime, days: number): TimeRange =>
{
    const to = Date.parse(lastReadingAt) + HOUR_MS;

    return {from: new Date(to - days * DAY_MS).toISOString(), to: new Date(to).toISOString()};
}

export const splitAt = (readings: Reading[], changeAt: IsoDateTime | null): ReadingWindows =>
{
    if (!changeAt) return {before: readings, after: []};

    const cut = Date.parse(changeAt);

    return {
        before: readings.filter(reading => Date.parse(reading.timestamp) < cut),
        after: readings.filter(reading => Date.parse(reading.timestamp) >= cut),
    };
}

export const average = (readings: Reading[], field: ReadingField): number | null =>
{
    if (readings.length === 0) return null;

    return readings.reduce((sum, reading) => sum + reading[field], 0) / readings.length;
}

export const extreme = (readings: Reading[], field: ReadingField, pick: 'min' | 'max'): number | null =>
{
    if (readings.length === 0) return null;

    const values = readings.map(reading => reading[field]);

    return pick === 'min' ? Math.min(...values) : Math.max(...values);
}

export const changePct = (before: number | null, after: number | null): number | null =>
{
    if (before === null || after === null || before === 0) return null;

    return ((after - before) / before) * 100;
}
