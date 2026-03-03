import { writable } from 'svelte/store';

export interface AuthUser {
	id: number;
	username: string;
	role: 'user' | 'admin';
}

export interface AuthState {
	accessToken: string | null;
	user: AuthUser | null;
	expiresAt: number | null;
}

export const auth = writable<AuthState>({
	accessToken: null,
	user: null,
	expiresAt: null
});
