import { get } from 'svelte/store';
import { auth } from '$lib/stores/auth';
import { goto } from '$app/navigation';
import { openSyncQueue } from '$lib/db/idb';

let proactiveTimer: ReturnType<typeof setTimeout> | null = null;
let refreshPromise: Promise<void> | null = null;

function postTokenToSW(token: string | null): void {
	if (!('serviceWorker' in navigator)) return;
	navigator.serviceWorker.ready.then((reg) => {
		reg.active?.postMessage({ type: 'SET_ACCESS_TOKEN', token });
	}).catch(() => undefined);
}

export function setTokens(accessToken: string, expiresAt: number, user: { id: number; username: string; role: 'user' | 'admin' }) {
	auth.set({ accessToken, user, expiresAt });
	scheduleProactiveRefresh(expiresAt);
	postTokenToSW(accessToken);
}

export function clearTokens() {
	auth.set({ accessToken: null, user: null, expiresAt: null });
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

export async function signOut(): Promise<void> {
	try {
		await apiFetch('/api/auth/logout', { method: 'POST' });
	} catch {
		// ignore network/logout errors; local session still needs to be cleared
	}
	clearTokens();
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
		const res = await fetch('/api/auth/refresh', {
			method: 'POST',
			credentials: 'include'
		});
		if (!res.ok) throw new Error('refresh failed');
		const data = await res.json();
		const user = get(auth).user;
		if (!user) throw new Error('no user in store');
		setTokens(data.access_token, Date.now() + data.expires_in * 1000, user);
	})();
	try {
		await refreshPromise;
	} finally {
		refreshPromise = null;
	}
}

export async function restoreSession(): Promise<void> {
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
