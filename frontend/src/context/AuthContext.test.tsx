import {act, renderHook} from '@testing-library/react';
import type {ReactNode} from 'react';
import AuthProvider from '@/context/AuthContext.tsx';
import {EXPIRY_WARNING_MS, useAuth} from '@/context/auth.ts';
import type {Session} from '@/interfaces/auth.ts';
import {ApiError} from '@/lib/api.ts';
import {clearSession, notifySessionExpired, readSession, saveSession, SESSION_STORAGE_KEY} from '@/lib/session.ts';

const NOW = new Date('2026-09-27T22:00:00Z');
const admin = {id: 'f90768ae-b4ed-425e-bdbd-918714c97484', email: 'admin@bia.app', name: 'Admin BIA'};

const wrapper = ({children}: {children: ReactNode}) => <AuthProvider>{children}</AuthProvider>;

const sessionExpiringIn = (milliseconds: number): Session => ({
    token: 'valid.jwt.token',
    expiresAt: new Date(NOW.getTime() + milliseconds).toISOString(),
    user: admin,
});

const jsonResponse = (status: number, body: unknown) =>
    new Response(JSON.stringify(body), {status, headers: {'Content-Type': 'application/json'}});

describe('AuthProvider', () =>
{
    const fetchMock = vi.fn();

    beforeEach(() =>
    {
        vi.useFakeTimers();
        vi.setSystemTime(NOW);
        clearSession();
        fetchMock.mockReset();
        vi.stubGlobal('fetch', fetchMock);
    });

    afterEach(() =>
    {
        vi.useRealTimers();
        vi.unstubAllGlobals();
    });

    it('restores a stored session on mount', () =>
    {
        saveSession(sessionExpiringIn(60 * 60 * 1000));

        const {result} = renderHook(() => useAuth(), {wrapper});

        expect(result.current.isAuthenticated).toBe(true);
        expect(result.current.user).toEqual(admin);
    });

    it('logs in, stores the session and exposes the user', async () =>
    {
        const expiresAt = new Date(NOW.getTime() + 12 * 60 * 60 * 1000).toISOString();
        fetchMock.mockResolvedValue(jsonResponse(200, {token: 'new.jwt.token', tokenType: 'Bearer', expiresAt, user: admin}));

        const {result} = renderHook(() => useAuth(), {wrapper});

        await act(async () =>
        {
            await result.current.login({email: 'admin@bia.app', password: 'secret-password'});
        });

        expect(result.current.isAuthenticated).toBe(true);
        expect(result.current.expiresAt).toBe(expiresAt);
        expect(readSession()?.token).toBe('new.jwt.token');
    });

    it('rejects wrong credentials without flagging an expired session', async () =>
    {
        fetchMock.mockResolvedValue(jsonResponse(401, {error: {code: 'INVALID_CREDENTIALS', message: 'Email or password is incorrect'}}));

        const {result} = renderHook(() => useAuth(), {wrapper});

        let error: unknown;
        await act(async () =>
        {
            error = await result.current.login({email: 'admin@bia.app', password: 'wrong'}).catch(e => e);
        });

        expect(error).toBeInstanceOf(ApiError);
        expect((error as ApiError).message).toBe('Correo o contraseña incorrectos.');
        expect(result.current.isAuthenticated).toBe(false);
        expect(result.current.sessionExpired).toBe(false);
    });

    it('warns five minutes before expiring and logs out when the token expires', () =>
    {
        saveSession(sessionExpiringIn(10 * 60 * 1000));

        const {result} = renderHook(() => useAuth(), {wrapper});

        act(() => vi.advanceTimersByTime(10 * 60 * 1000 - EXPIRY_WARNING_MS - 1));
        expect(result.current.isExpiringSoon).toBe(false);

        act(() => vi.advanceTimersByTime(1));
        expect(result.current.isExpiringSoon).toBe(true);

        act(() => vi.advanceTimersByTime(EXPIRY_WARNING_MS));
        expect(result.current.isAuthenticated).toBe(false);
        expect(result.current.sessionExpired).toBe(true);
        expect(readSession()).toBeNull();
    });

    it('logs out when any request reports an expired token', () =>
    {
        saveSession(sessionExpiringIn(60 * 60 * 1000));

        const {result} = renderHook(() => useAuth(), {wrapper});

        act(() => notifySessionExpired());

        expect(result.current.isAuthenticated).toBe(false);
        expect(result.current.sessionExpired).toBe(true);
    });

    it('logs out on purpose without showing the expired notice', () =>
    {
        saveSession(sessionExpiringIn(60 * 60 * 1000));

        const {result} = renderHook(() => useAuth(), {wrapper});

        act(() => result.current.logout());

        expect(result.current.isAuthenticated).toBe(false);
        expect(result.current.sessionExpired).toBe(false);
        expect(readSession()).toBeNull();
    });

    it('follows a logout made in another tab', () =>
    {
        saveSession(sessionExpiringIn(60 * 60 * 1000));

        const {result} = renderHook(() => useAuth(), {wrapper});

        clearSession();
        act(() => window.dispatchEvent(new StorageEvent('storage', {key: SESSION_STORAGE_KEY})));

        expect(result.current.isAuthenticated).toBe(false);
    });

    it('fails loudly when used outside the provider', () =>
    {
        vi.spyOn(console, 'error').mockImplementation(() => undefined);

        expect(() => renderHook(() => useAuth())).toThrow('useAuth must be used within an AuthProvider');
    });
});
