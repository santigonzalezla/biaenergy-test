type Reply = unknown | ((request: {method: string; path: string; body: unknown}) => unknown);

export interface MockRoute {
    status?: number;
    reply: Reply;
}

export type MockRoutes = Record<string, MockRoute | MockRoute[]>;

export const apiError = (status: number, code: string, message = code): MockRoute => ({
    status,
    reply: {error: {code, message, path: '', timestamp: '2026-09-27T22:00:00Z'}},
});

export const ok = (reply: Reply, status = 200): MockRoute => ({status, reply});

export const mockApi = (routes: MockRoutes) =>
{
    const calls = new Map<string, number>();

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) =>
    {
        const url = new URL(String(input));
        const method = init?.method ?? 'GET';
        const path = url.pathname.replace(/^\/api/, '') + url.search;
        const key = `${method} ${path}`;
        const entry = routes[key] ?? routes[`${method} ${url.pathname.replace(/^\/api/, '')}`];

        if (!entry) throw new Error(`No mock for ${key}`);

        const count = calls.get(key) ?? 0;
        calls.set(key, count + 1);

        const route = Array.isArray(entry) ? entry[Math.min(count, entry.length - 1)] : entry;
        const body = typeof init?.body === 'string' ? JSON.parse(init.body) : undefined;
        const payload = typeof route.reply === 'function' ? route.reply({method, path, body}) : route.reply;

        return new Response(JSON.stringify(payload), {status: route.status ?? 200, headers: {'Content-Type': 'application/json'}});
    });

    vi.stubGlobal('fetch', fetchMock);

    return fetchMock;
}
