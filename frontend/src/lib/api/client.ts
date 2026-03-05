import { get } from 'svelte/store';
import { auth } from '$lib/stores/auth';
import { goto } from '$app/navigation';
import { openSyncQueue } from '$lib/db/idb';

const REFRESH_KEY = 'mm_refresh_token';
let proactiveTimer: ReturnType<typeof setTimeout> | null = null;
let refreshPromise: Promise<void> | null = null;

function postTokenToSW(token: string | null): void {
	if (!('serviceWorker' in navigator)) return;
	navigator.serviceWorker.ready.then((reg) => {
		reg.active?.postMessage({ type: 'SET_ACCESS_TOKEN', token });
	}).catch(() => undefined);
}

export function setTokens(accessToken: string, refreshToken: string, expiresAt: number, user: { id: number; username: string; role: 'user' | 'admin' }) {
	auth.set({ accessToken, user, expiresAt });
	localStorage.setItem(REFRESH_KEY, refreshToken);
	scheduleProactiveRefresh(expiresAt);
	postTokenToSW(accessToken);
}

export function clearTokens() {
	auth.set({ accessToken: null, user: null, expiresAt: null });
	localStorage.removeItem(REFRESH_KEY);
	if (proactiveTimer !== null) {
		clearTimeout(proactiveTimer);
		proactiveTimer = null;
	}
	postTokenToSW(null);
	openSyncQueue().then((db) => db.clear('sync_queue')).catch(() => undefined);
	if ('caches' in window) {
		caches.delete('api-competitions').catch(() => undefined);
	}
}

function scheduleProactiveRefresh(expiresAt: number) {
	if (proactiveTimer !== null) clearTimeout(proactiveTimer);
	const delay = expiresAt - Date.now() - 5 * 60 * 1000;
	if (delay > 0) {
		proactiveTimer = setTimeout(() => {
			silentRefresh().catch(() => {
				clearTokens();
				goto('/login');
			});
		}, delay);
	}
}

export async function silentRefresh(): Promise<void> {
	if (refreshPromise) return refreshPromise;
	refreshPromise = (async () => {
		const refreshToken = localStorage.getItem(REFRESH_KEY);
		if (!refreshToken) throw new Error('no refresh token');
		const res = await fetch('/api/auth/refresh', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refreshToken })
		});
		if (!res.ok) throw new Error('refresh failed');
		const data = await res.json();
		setTokens(data.access_token, data.refresh_token, Date.now() + data.expires_in * 1000, get(auth).user!);
	})();
	try {
		await refreshPromise;
	} finally {
		refreshPromise = null;
	}
}

export async function restoreSession(): Promise<void> {
	const refreshToken = localStorage.getItem(REFRESH_KEY);
	if (!refreshToken) return;
	try {
		await silentRefresh();
	} catch {
		clearTokens();
	}
}

export async function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
	const state = get(auth);
	const headers = new Headers(init.headers);
	if (state.accessToken) {
		headers.set('Authorization', `Bearer ${state.accessToken}`);
	}
	if (!headers.has('Content-Type') && init.body) {
		headers.set('Content-Type', 'application/json');
	}

	let res = await fetch(path, { ...init, headers });

	if (res.status === 401) {
		try {
			await silentRefresh();
			const refreshed = get(auth);
			if (refreshed.accessToken) {
				headers.set('Authorization', `Bearer ${refreshed.accessToken}`);
			}
			res = await fetch(path, { ...init, headers });
		} catch {
			clearTokens();
			goto('/login');
		}
	}

	return res;
}
