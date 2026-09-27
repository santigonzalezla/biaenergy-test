import {useCallback, useEffect, useRef, useState} from 'react';
import {ApiError, apiRequest, type HttpMethod} from '@/lib/api.ts';

interface UseFetchOptions {
    method?: HttpMethod;
    body?: unknown;
    immediate?: boolean;
}

interface ExecuteOptions {
    path?: string;
    method?: HttpMethod;
    body?: unknown;
}

interface UseFetchState<T> {
    data: T | null;
    isLoading: boolean;
    error: ApiError | null;
}

interface UseFetchResult<T> extends UseFetchState<T> {
    execute: (options?: ExecuteOptions) => Promise<T | null>;
    reset: () => void;
}

export const useFetch = <T>(path: string | null, options: UseFetchOptions = {}): UseFetchResult<T> =>
{
    const {method = 'GET', body, immediate = method === 'GET'} = options;
    const shouldRun = immediate && path !== null;
    const bodyKey = body === undefined ? undefined : JSON.stringify(body);
    const controllerRef = useRef<AbortController | null>(null);

    const [state, setState] = useState<UseFetchState<T>>({data: null, isLoading: shouldRun, error: null});

    const execute = useCallback(async (overrides: ExecuteOptions = {}): Promise<T | null> =>
    {
        const target = overrides.path ?? path;

        if (target === null) return null;

        controllerRef.current?.abort();
        const controller = new AbortController();
        controllerRef.current = controller;

        setState(previous => ({...previous, isLoading: true, error: null}));

        try
        {
            const data = await apiRequest<T>(target, {
                method: overrides.method ?? method,
                body: overrides.body ?? (bodyKey === undefined ? undefined : JSON.parse(bodyKey)),
                signal: controller.signal,
            });

            if (!controller.signal.aborted) setState({data, isLoading: false, error: null});

            return data;
        }
        catch (error)
        {
            if (controller.signal.aborted) return null;

            setState(previous => ({...previous, isLoading: false, error: ApiError.from(error)}));

            return null;
        }
    }, [path, method, bodyKey]);

    const reset = useCallback(() =>
    {
        controllerRef.current?.abort();
        setState({data: null, isLoading: false, error: null});
    }, []);

    useEffect(() =>
    {
        if (shouldRun) void execute();

        return () => controllerRef.current?.abort();
    }, [execute, shouldRun]);

    return {...state, execute, reset};
}

export default useFetch;
