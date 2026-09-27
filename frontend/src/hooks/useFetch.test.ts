import {act, renderHook, waitFor} from '@testing-library/react';
import {useFetch} from '@/hooks/useFetch.ts';

const jsonResponse = (status: number, body: unknown) =>
    new Response(JSON.stringify(body), {status, headers: {'Content-Type': 'application/json'}});

describe('useFetch', () =>
{
    const fetchMock = vi.fn();

    beforeEach(() =>
    {
        fetchMock.mockReset();
        vi.stubGlobal('fetch', fetchMock);
    });

    afterEach(() => vi.unstubAllGlobals());

    it('loads GET requests immediately', async () =>
    {
        fetchMock.mockResolvedValue(jsonResponse(200, {metersCount: 12}));

        const {result} = renderHook(() => useFetch<{metersCount: number}>('/dashboard/summary'));

        expect(result.current.isLoading).toBe(true);
        await waitFor(() => expect(result.current.isLoading).toBe(false));
        expect(result.current.data).toEqual({metersCount: 12});
        expect(result.current.error).toBeNull();
    });

    it('waits for execute on POST requests', async () =>
    {
        fetchMock.mockResolvedValue(jsonResponse(202, {id: 'analysis-1'}));

        const {result} = renderHook(() => useFetch<{id: string}>('/ai/analyze', {method: 'POST'}));

        expect(result.current.isLoading).toBe(false);
        expect(fetchMock).not.toHaveBeenCalled();

        let returned: {id: string} | null = null;
        await act(async () =>
        {
            returned = await result.current.execute();
        });

        expect(returned).toEqual({id: 'analysis-1'});
        expect(result.current.data).toEqual({id: 'analysis-1'});
    });

    it('does not request anything while the path is null', () =>
    {
        const {result} = renderHook(() => useFetch(null));

        expect(result.current.isLoading).toBe(false);
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it('exposes the ApiError and keeps the previous data', async () =>
    {
        fetchMock.mockResolvedValueOnce(jsonResponse(200, {data: [1]}));
        const {result} = renderHook(() => useFetch<{data: number[]}>('/anomalies'));
        await waitFor(() => expect(result.current.data).toEqual({data: [1]}));

        fetchMock.mockResolvedValueOnce(jsonResponse(500, {error: {code: 'INTERNAL_ERROR', message: 'boom'}}));
        await act(async () =>
        {
            await result.current.execute();
        });

        expect(result.current.error?.code).toBe('INTERNAL_ERROR');
        expect(result.current.error?.message).toBe('Ocurrió un error en el servidor, intenta de nuevo.');
        expect(result.current.data).toEqual({data: [1]});
    });

    it('refetches when the path changes and aborts the previous request', async () =>
    {
        fetchMock.mockImplementation(() => new Promise(() => {}));

        const {rerender} = renderHook(({path}) => useFetch(path), {initialProps: {path: '/anomalies?type=REAL_ANOMALY'}});
        const firstSignal: AbortSignal = fetchMock.mock.calls[0][1].signal;

        rerender({path: '/anomalies?type=DATA_QUALITY'});

        expect(firstSignal.aborted).toBe(true);
        expect(fetchMock).toHaveBeenCalledTimes(2);
        expect(fetchMock.mock.calls[1][0]).toContain('type=DATA_QUALITY');
    });

    it('aborts the pending request on unmount', () =>
    {
        fetchMock.mockImplementation(() => new Promise(() => {}));

        const {unmount} = renderHook(() => useFetch('/meters'));
        const signal: AbortSignal = fetchMock.mock.calls[0][1].signal;

        unmount();

        expect(signal.aborted).toBe(true);
    });
});
