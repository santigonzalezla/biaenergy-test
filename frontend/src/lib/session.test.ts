import type {Session} from '@/interfaces/auth.ts';
import {clearSession, notifySessionExpired, onSessionExpired, readSession, saveSession} from '@/lib/session.ts';

const session = (expiresAt: string): Session => ({
    token: 'header.payload.signature',
    expiresAt,
    user: {id: 'f90768ae-b4ed-425e-bdbd-918714c97484', email: 'admin@bia.app', name: 'Admin BIA'},
});

describe('session', () =>
{
    beforeEach(() => clearSession());

    it('returns a saved session that has not expired', () =>
    {
        const valid = session(new Date(Date.now() + 60_000).toISOString());
        saveSession(valid);

        expect(readSession()).toEqual(valid);
    });

    it('discards an expired session', () =>
    {
        saveSession(session(new Date(Date.now() - 1_000).toISOString()));

        expect(readSession()).toBeNull();
        expect(localStorage.getItem('biaenergy.session')).toBeNull();
    });

    it('discards a corrupted session', () =>
    {
        localStorage.setItem('biaenergy.session', '{not json');

        expect(readSession()).toBeNull();
    });

    it('clears the session and notifies every listener when it expires', () =>
    {
        saveSession(session(new Date(Date.now() + 60_000).toISOString()));
        const listener = vi.fn();
        const unsubscribe = onSessionExpired(listener);

        notifySessionExpired();
        unsubscribe();
        notifySessionExpired();

        expect(readSession()).toBeNull();
        expect(listener).toHaveBeenCalledTimes(1);
    });
});
