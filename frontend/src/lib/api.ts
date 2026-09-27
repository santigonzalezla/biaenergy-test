import type {ApiErrorBody, RowError} from '@/interfaces/interfaces.ts';
import {API_BASE_URL} from '@/lib/constants.ts';
import {notifySessionExpired, readSession} from '@/lib/session.ts';

export type HttpMethod = 'GET' | 'POST' | 'PATCH' | 'DELETE';

export type QueryValue = string | number | boolean | null | undefined;

export interface RequestOptions {
    method?: HttpMethod;
    body?: unknown;
    signal?: AbortSignal;
}

const SESSION_ERROR_CODES = new Set(['MISSING_TOKEN', 'INVALID_TOKEN', 'TOKEN_EXPIRED']);

const ERROR_MESSAGES: Record<string, string> = {
    INVALID_CREDENTIALS: 'Correo o contraseña incorrectos.',
    TOKEN_EXPIRED: 'Tu sesión expiró, vuelve a ingresar.',
    INVALID_TOKEN: 'Tu sesión no es válida, vuelve a ingresar.',
    MISSING_TOKEN: 'Debes iniciar sesión para continuar.',
    VALIDATION_ERROR: 'Revisa los datos enviados.',
    NETWORK_ERROR: 'No se pudo conectar con el servidor.',
    INTERNAL_ERROR: 'Ocurrió un error en el servidor, intenta de nuevo.',
    UNEXPECTED_RESPONSE: 'El servidor respondió de forma inesperada.',
};

export class ApiError extends Error
{
    readonly status: number;
    readonly code: string;
    readonly details?: Record<string, string> | RowError[];

    constructor(status: number, code: string, message: string, details?: Record<string, string> | RowError[])
    {
        super(ERROR_MESSAGES[code] ?? message);
        this.name = 'ApiError';
        this.status = status;
        this.code = code;
        this.details = details;
    }

    static from(error: unknown): ApiError
    {
        if (error instanceof ApiError) return error;

        return new ApiError(0, 'NETWORK_ERROR', error instanceof Error ? error.message : 'Network error');
    }

    get isSessionError(): boolean
    {
        return this.status === 401 && SESSION_ERROR_CODES.has(this.code);
    }
}

export const buildQuery = (params: Record<string, QueryValue> = {}): string =>
{
    const search = new URLSearchParams();

    Object.entries(params).forEach(([key, value]) =>
    {
        if (value !== undefined && value !== null && value !== '') search.set(key, String(value));
    });

    const query = search.toString();

    return query ? `?${query}` : '';
}

export const apiRequest = async <T>(path: string, options: RequestOptions = {}): Promise<T> =>
{
    const {method = 'GET', body, signal} = options;
    const isFormData = body instanceof FormData;
    const headers: Record<string, string> = {Accept: 'application/json'};
    const session = readSession();

    if (session) headers.Authorization = `Bearer ${session.token}`;

    if (body !== undefined && !isFormData) headers['Content-Type'] = 'application/json';

    const response = await fetch(`${API_BASE_URL}/api${path}`, {
        method,
        headers,
        signal,
        body: body === undefined ? undefined : isFormData ? body : JSON.stringify(body),
    });

    if (response.status === 204) return undefined as T;

    const payload = await response.json().catch(() => null);

    if (response.ok) return payload as T;

    const apiError = toApiError(response.status, payload);

    if (apiError.isSessionError) notifySessionExpired();

    throw apiError;
}

const toApiError = (status: number, payload: unknown): ApiError =>
{
    const error = (payload as ApiErrorBody | null)?.error;

    if (!error?.code) return new ApiError(status, 'UNEXPECTED_RESPONSE', `Unexpected response with status ${status}`);

    return new ApiError(status, error.code, error.message, error.details);
}
