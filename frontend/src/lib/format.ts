const LOCALE = 'es-CO';

export const APP_TIMEZONE = 'America/Bogota';

const decimal = (digits: number) => new Intl.NumberFormat(LOCALE, {minimumFractionDigits: 0, maximumFractionDigits: digits});

const dateTime = new Intl.DateTimeFormat(LOCALE, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: APP_TIMEZONE,
});

const shortDateTime = new Intl.DateTimeFormat(LOCALE, {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: APP_TIMEZONE,
});

export const formatNumber = (value: number, digits: number = 0) => decimal(digits).format(value);

export const formatKwh = (value: number, digits: number = 2) => `${formatNumber(value, digits)} kWh`;

export const formatPercent = (value: number, digits: number = 1) =>
{
    const sign = value > 0 ? '+' : '';

    return `${sign}${formatNumber(value, digits)} %`;
}

export const formatRatio = (value: number) => `${formatNumber(value * 100, 0)} %`;

export const formatDateTime = (iso: string) => dateTime.format(new Date(iso));

export const formatShortDateTime = (iso: string) => shortDateTime.format(new Date(iso));

export const formatDuration = (milliseconds: number) =>
{
    const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000));
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;

    return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}
