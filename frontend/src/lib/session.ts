import type {Session} from '@/interfaces/auth.ts';

export const SESSION_STORAGE_KEY = 'biaenergy.session';

type Listener = () => void;

const expiredListeners = new Set<Listener>();

export const readSession = (): Session | null =>
{
    try
    {
        const raw = localStorage.getItem(SESSION_STORAGE_KEY);

        if (!raw) return null;

        const session = JSON.parse(raw) as Session;

        if (!session.token || isExpired(session))
        {
            clearSession();
            return null;
        }

        return session;
    }
    catch
    {
        clearSession();
        return null;
    }
}

export const saveSession = (session: Session) =>
{
    localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(session));
}

export const clearSession = () =>
{
    localStorage.removeItem(SESSION_STORAGE_KEY);
}

export const isExpired = (session: Session, now: number = Date.now()) =>
{
    const expiresAt = Date.parse(session.expiresAt);

    return Number.isNaN(expiresAt) || expiresAt <= now;
}

export const onSessionExpired = (listener: Listener) =>
{
    expiredListeners.add(listener);

    return () =>
    {
        expiredListeners.delete(listener);
    };
}

export const notifySessionExpired = () =>
{
    clearSession();
    expiredListeners.forEach(listener => listener());
}
