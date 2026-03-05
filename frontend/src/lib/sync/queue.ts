import { openSyncQueue, type SyncQueueItem } from '$lib/db/idb';
import { get } from 'svelte/store';
import { auth } from '$lib/stores/auth';

const MAX_RETRIES = 3;

export async function enqueue(item: Omit<SyncQueueItem, 'id' | 'retries' | 'errored' | 'timestamp'>): Promise<void> {
	const db = await openSyncQueue();
	await db.add('sync_queue', {
		...item,
		retries: 0,
		errored: false,
		timestamp: Date.now()
	});
}

export async function replayQueue(): Promise<void> {
	const db = await openSyncQueue();
	const items = await db.getAll('sync_queue');
	const token = get(auth).accessToken;

	for (const item of items) {
		if (item.errored) continue;
		try {
		const res = await fetch(
			`/api/competitions/${item.competitionId}/lanes/${item.lane}/stage/${item.stage}/hits`,
			{
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					...(token ? { Authorization: `Bearer ${token}` } : {})
				},
				body: JSON.stringify({ hits: item.hits }),
				signal: AbortSignal.timeout(10_000)
			}
		);
			if (res.ok) {
				await db.delete('sync_queue', item.id as number);
			} else {
				await markRetry(db, item);
			}
		} catch {
			await markRetry(db, item);
		}
	}
}

async function markRetry(db: Awaited<ReturnType<typeof openSyncQueue>>, item: SyncQueueItem): Promise<void> {
	item.retries += 1;
	if (item.retries >= MAX_RETRIES) {
		item.errored = true;
	}
	await db.put('sync_queue', item);
}

export async function registerBackgroundSync(): Promise<void> {
	if (!('serviceWorker' in navigator) || !('SyncManager' in window)) return;
	const reg = await navigator.serviceWorker.ready;
	await reg.sync.register('mm-sync');
}

export function installIOSFallback(): () => void {
	const handler = () => {
		if (document.visibilityState === 'visible') {
			replayQueue().catch(() => undefined);
		}
	};
	document.addEventListener('visibilitychange', handler);
	return () => document.removeEventListener('visibilitychange', handler);
}
