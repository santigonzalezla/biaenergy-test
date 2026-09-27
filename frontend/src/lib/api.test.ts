import {ApiError, apiRequest, buildQuery} from '@/lib/api.ts';
import {clearSession, onSessionExpired, saveSession} from '@/lib/session.ts';

const jsonResponse = (status: number, body: unknown) =>
    new Response(JSON.stringify(body), {status, headers: {'Content-Type': 'application/json'}});

const errorResponse = (status: number, code: string, message: string) =>
    jsonResponse(status, {error: {code, message, path: '/api/x', timestamp: '2026-09-27T22:00:00Z'}});

const saveValidSession = () => saveSession({
    token: 'valid.jwt.token',
    expiresAt: new Date(Date.now() + 60_000).toISOString(),
    user: {id: '1', email: 'admin@bia.app', name: 'Admin BIA'},
});

describe('apiRequest', () =>
{
    const fetchMock = vi.fn();

    beforeEach(() =>
    {
        clearSession();
        fetchMock.mockReset();
        vi.stubGlobal('fetch', fetchMock);
    });

    afterEach(() => vi.unstubAllGlobals());

    it('prefixes /api and sends the bearer token when there is a session', async () =>
    {
        saveValidSession();
        fetchMock.mockResolvedValue(jsonResponse(200, {status: 'ok'}));

        await expect(apiRequest('/health')).resolves.toEqual({status: 'ok'});

        const [url, init] = fetchMock.mock.calls[0];
        expect(url).toBe('http://localhost:8080/api/health');
        expect(init.headers.Authorization).toBe('Bearer valid.jwt.token');
    });

    it('serializes JSON bodies but lets FormData set its own boundary', async () =>
    {
        fetchMock.mockResolvedValue(jsonResponse(200, {}));

        await apiRequest('/auth/login', {method: 'POST', body: {email: 'admin@bia.app'}});
        await apiRequest('/readings/import', {method: 'POST', body: new FormData()});

        expect(fetchMock.mock.calls[0][1].headers['Content-Type']).toBe('application/json');
        expect(fetchMock.mock.calls[0][1].body).toBe('{"email":"admin@bia.app"}');
        expect(fetchMock.mock.calls[1][1].headers['Content-Type']).toBeUndefined();
    });

    it('resolves undefined for 204 No Content', async () =>
    {
        fetchMock.mockResolvedValue(new Response(null, {status: 204}));

        await expect(apiRequest('/meters/1', {method: 'DELETE'})).resolves.toBeUndefined();
    });

    it('turns the Go error format into an ApiError with a Spanish message', async () =>
    {
        fetchMock.mockResolvedValue(errorResponse(404, 'METER_NOT_FOUND', 'Meter not found'));

        const error = await apiRequest('/meters/1').catch(e => e);

        expect(error).toBeInstanceOf(ApiError);
        expect(error).toMatchObject({status: 404, code: 'METER_NOT_FOUND', message: 'Meter not found'});
    });

    it('expires the session on TOKEN_EXPIRED but not on INVALID_CREDENTIALS', async () =>
    {
        const listener = vi.fn();
        const unsubscribe = onSessionExpired(listener);

        fetchMock.mockResolvedValueOnce(errorResponse(401, 'INVALID_CREDENTIALS', 'Email or password is incorrect'));
        const loginError = await apiRequest<never>('/auth/login', {method: 'POST', body: {}}).catch((e: ApiError) => e);

        fetchMock.mockResolvedValueOnce(errorResponse(401, 'TOKEN_EXPIRED', 'The session has expired'));
        const sessionError = await apiRequest<never>('/meters').catch((e: ApiError) => e);

        unsubscribe();

        expect(loginError.message).toBe('Correo o contraseña incorrectos.');
        expect(sessionError.message).toBe('Tu sesión expiró, vuelve a ingresar.');
        expect(listener).toHaveBeenCalledTimes(1);
    });

    it('reports a non JSON error body as UNEXPECTED_RESPONSE', async () =>
    {
        fetchMock.mockResolvedValue(new Response('<html>Bad Gateway</html>', {status: 502}));

        await expect(apiRequest('/meters')).rejects.toMatchObject({status: 502, code: 'UNEXPECTED_RESPONSE'});
    });
});

describe('buildQuery', () =>
{
    it('skips empty values and encodes the rest', () =>
    {
        expect(buildQuery({search: 'M-109', status: undefined, page: 1, type: '', note: 'a b'}))
            .toBe('?search=M-109&page=1&note=a+b');
    });

    it('returns an empty string without params', () =>
    {
        expect(buildQuery({status: null})).toBe('');
    });
});
