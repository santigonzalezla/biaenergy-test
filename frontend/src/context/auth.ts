import {createContext, useContext} from 'react';
import type {AuthUser, LoginRequest} from '@/interfaces/auth.ts';

export const EXPIRY_WARNING_MS = 5 * 60 * 1000;

export interface AuthContextType {
    user: AuthUser | null;
    expiresAt: string | null;
    isAuthenticated: boolean;
    isExpiringSoon: boolean;
    sessionExpired: boolean;
    login: (credentials: LoginRequest) => Promise<AuthUser>;
    logout: () => void;
    dismissExpiredNotice: () => void;
}

export const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () =>
{
    const context = useContext(AuthContext);

    if (context === undefined) throw new Error('useAuth must be used within an AuthProvider');

    return context;
}
