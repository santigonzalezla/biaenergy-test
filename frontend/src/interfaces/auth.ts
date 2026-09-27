import type {IsoDateTime} from '@/interfaces/interfaces.ts';

export interface AuthUser {
    id: string;
    email: string;
    name: string;
}

export interface LoginRequest {
    email: string;
    password: string;
}

export interface LoginResponse {
    token: string;
    tokenType: 'Bearer';
    expiresAt: IsoDateTime;
    user: AuthUser;
}

export interface Session {
    token: string;
    expiresAt: IsoDateTime;
    user: AuthUser;
}
