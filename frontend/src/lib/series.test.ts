import type {Reading} from '@/interfaces/interfaces.ts';
import {average, changePct, extreme, rangeEndingAt, splitAt} from '@/lib/series.ts';

const reading = (timestamp: string, consumptionKwh: number, powerFactor = 0.94): Reading => ({
    timestamp, consumptionKwh, voltage: 220, current: 240, powerFactor, status: 'OK',
});

const readings = [
    reading('2026-09-12T17:00:00Z', 42),
    reading('2026-09-12T18:00:00Z', 44),
    reading('2026-09-12T19:00:00Z', 92, 0.72),
    reading('2026-09-12T20:00:00Z', 94, 0.70),
];

describe('series', () =>
{
    it('builds a range that ends one hour after the last reading', () =>
    {
        expect(rangeEndingAt('2026-09-15T04:00:00Z', 14)).toEqual({
            from: '2026-09-01T05:00:00.000Z',
            to: '2026-09-15T05:00:00.000Z',
        });
    });

    it('splits the readings at the change point, which belongs to the after window', () =>
    {
        const {before, after} = splitAt(readings, '2026-09-12T19:00:00Z');

        expect(before.map(item => item.consumptionKwh)).toEqual([42, 44]);
        expect(after.map(item => item.consumptionKwh)).toEqual([92, 94]);
    });

    it('keeps every reading before when there is no change point', () =>
    {
        expect(splitAt(readings, null)).toEqual({before: readings, after: []});
    });

    it('computes averages, extremes and the change between windows', () =>
    {
        const {before, after} = splitAt(readings, '2026-09-12T19:00:00Z');

        expect(average(before, 'consumptionKwh')).toBe(43);
        expect(average(after, 'consumptionKwh')).toBe(93);
        expect(extreme(after, 'powerFactor', 'min')).toBe(0.70);
        expect(changePct(43, 93)).toBeCloseTo(116.28, 2);
        expect(average([], 'voltage')).toBeNull();
        expect(changePct(null, 10)).toBeNull();
    });
});
