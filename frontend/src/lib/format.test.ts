import {formatDateTime, formatDuration, formatKwh, formatNumber, formatPercent, formatRatio, formatShortDateTime} from '@/lib/format.ts';

describe('format', () =>
{
    it('uses Colombian separators for numbers and energy', () =>
    {
        expect(formatNumber(4032)).toBe('4.032');
        expect(formatKwh(155250.85)).toBe('155.250,85 kWh');
    });

    it('signs percentages and turns ratios into percentages', () =>
    {
        expect(formatPercent(110.5)).toBe('+110,5 %');
        expect(formatPercent(-79.8)).toBe('-79,8 %');
        expect(formatPercent(0)).toBe('0 %');
        expect(formatRatio(0.9)).toBe('90 %');
    });

    it('shows UTC timestamps in Bogotá time', () =>
    {
        expect(formatDateTime('2026-09-12T19:00:00Z')).toBe('12 de sept de 2026, 14:00');
        expect(formatShortDateTime('2026-09-27T22:13:30Z')).toBe('27 de sept, 17:13');
    });

    it('formats elapsed time as mm:ss', () =>
    {
        expect(formatDuration(14_900)).toBe('00:14');
        expect(formatDuration(81_000)).toBe('01:21');
        expect(formatDuration(-5)).toBe('00:00');
    });
});
