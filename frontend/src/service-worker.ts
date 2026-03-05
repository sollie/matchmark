/// <reference lib="webworker" />
/// <reference types="@vite-pwa/sveltekit/client" />
import { cleanupOutdatedCaches, precacheAndRoute } from 'workbox-precaching';
import { NetworkFirst } from 'workbox-strategies';
import { registerRoute } from 'workbox-routing';
import { openSyncQueue } from '$lib/db/idb';

declare const self: ServiceWorkerGlobalScope;

const MAX_RETRIES = 3;

let accessToken: string | null = null;

self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));

self.addEventListener('message', (e) => {
	if (e.origin !== self.location.origin) return;
	if (e.data?.type === 'SET_ACCESS_TOKEN') {
		accessToken = e.data.token ?? null;
	}
});

cleanupOutdatedCaches();

precacheAndRoute(self.__WB_MANIFEST);

registerRoute(
	({ url }) => url.pathname.startsWith('/api/competitions'),
	new NetworkFirst({ cacheName: 'api-competitions' })
);

self.addEventListener('sync', (e) => {
	if (e.tag === 'mm-sync') {
		e.waitUntil(replayQueue());
	}
});

async function replayQueue(): Promise<void> {
	const db = await openSyncQueue();
	const items = await db.getAll('sync_queue');
	for (const item of items) {
		if (item.errored) continue;
		try {
			const headers: Record<string, string> = { 'Content-Type': 'application/json' };
			if (accessToken) {
				headers['Authorization'] = `Bearer ${accessToken}`;
			}
			const res = await fetch(
				`/api/competitions/${item.competitionId}/lanes/${item.lane}/stage/${item.stage}/hits`,
				{
					method: 'POST',
					headers,
					body: JSON.stringify({ hits: item.hits }),
					signal: AbortSignal.timeout(10_000)
				}
			);
			if (res.ok) {
				await db.delete('sync_queue', item.id);
			} else {
				item.retries += 1;
				if (item.retries >= MAX_RETRIES) {
					item.errored = true;
				}
				await db.put('sync_queue', item);
			}
		} catch {
			item.retries += 1;
			if (item.retries >= MAX_RETRIES) {
				item.errored = true;
			}
			await db.put('sync_queue', item);
		}
	}
}
