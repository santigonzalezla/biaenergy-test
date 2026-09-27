import {useCallback, useEffect, useMemo, useState, type ReactNode} from 'react';
import {AuthContext, EXPIRY_WARNING_MS, type AuthContextType} from '@/context/auth.ts';
import type {LoginRequest, LoginResponse, Session} from '@/interfaces/auth.ts';
import {apiRequest} from '@/lib/api.ts';
import {clearSession, notifySessionExpired, onSessionExpired, readSession, saveSession, SESSION_STORAGE_KEY} from '@/lib/session.ts';

const MAX_TIMEOUT_MS = 2_147_483_647;

interface AuthProviderProps {
    children: ReactNode;
}

const clampDelay = (milliseconds: number) => Math.min(Math.max(milliseconds, 0), MAX_TIMEOUT_MS);

const AuthProvider = ({children}: AuthProviderProps) =>
{
    const [session, setSession] = useState<Session | null>(() => readSession());
    const [isExpiringSoon, setIsExpiringSoon] = useState(false);
    const [sessionExpired, setSessionExpired] = useState(false);

    const endSession = useCallback((expired: boolean) =>
    {
        setSession(null);
        setIsExpiringSoon(false);
        setSessionExpired(expired);
    }, []);

    useEffect(() => onSessionExpired(() => endSession(true)), [endSession]);

    useEffect(() =>
    {
        if (!session) return;

        const remaining = Date.parse(session.expiresAt) - Date.now();
        const warningTimer = setTimeout(() => setIsExpiringSoon(true), clampDelay(remaining - EXPIRY_WARNING_MS));
        const expiryTimer = setTimeout(() => notifySessionExpired(), clampDelay(remaining));

        return () =>
        {
            clearTimeout(warningTimer);
            clearTimeout(expiryTimer);
        };
    }, [session]);

    useEffect(() =>
    {
        const syncAcrossTabs = (event: StorageEvent) =>
        {
            if (event.key !== SESSION_STORAGE_KEY) return;

            const current = readSession();
            setSession(current);

            if (!current) setIsExpiringSoon(false);
        };

        window.addEventListener('storage', syncAcrossTabs);

        return () => window.removeEventListener('storage', syncAcrossTabs);
    }, []);

    const login = useCallback(async (credentials: LoginRequest) =>
    {
        const response = await apiRequest<LoginResponse>('/auth/login', {method: 'POST', body: credentials});
        const next: Session = {token: response.token, expiresAt: response.expiresAt, user: response.user};

        saveSession(next);
        setSession(next);
        setIsExpiringSoon(false);
        setSessionExpired(false);

        return response.user;
    }, []);

    const logout = useCallback(() =>
    {
        clearSession();
        endSession(false);
    }, [endSession]);

    const dismissExpiredNotice = useCallback(() => setSessionExpired(false), []);

    const value = useMemo<AuthContextType>(() => ({
        user: session?.user ?? null,
        expiresAt: session?.expiresAt ?? null,
        isAuthenticated: session !== null,
        isExpiringSoon,
        sessionExpired,
        login,
        logout,
        dismissExpiredNotice,
    }), [session, isExpiringSoon, sessionExpired, login, logout, dismissExpiredNotice]);

    return (
        <AuthContext.Provider value={value}>
            {children}
        </AuthContext.Provider>
    );
}

export default AuthProvider;
