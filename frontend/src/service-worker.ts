/// <reference lib="webworker" />
/// <reference types="@vite-pwa/sveltekit/client" />
import { cleanupOutdatedCaches, precacheAndRoute } from 'workbox-precaching';
import { NetworkFirst } from 'workbox-strategies';
import { registerRoute } from 'workbox-routing';
import { openSyncQueue } from '$lib/db/idb';

declare const self: ServiceWorkerGlobalScope;

self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));

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
		try {
			const res = await fetch(
				`/api/competitions/${item.competitionId}/lanes/${item.lane}/stage/${item.stage}/hits`,
				{
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ hits: item.hits })
				}
			);
			if (res.ok) {
				await db.delete('sync_queue', item.id);
			} else {
				item.retries += 1;
				if (item.retries >= 3) {
					item.errored = true;
				}
				await db.put('sync_queue', item);
			}
		} catch {
			item.retries += 1;
			if (item.retries >= 3) {
				item.errored = true;
			}
			await db.put('sync_queue', item);
		}
	}
}
