import { openDB, type IDBPDatabase } from 'idb';

export interface SyncQueueItem {
	id?: number;
	competitionId: number;
	lane: number;
	stage: number;
	hits: string[];
	retries: number;
	errored: boolean;
	timestamp: number;
}

export interface MatchMarkDB {
	sync_queue: {
		key: number;
		value: SyncQueueItem;
		indexes: { by_competition: number };
	};
}

let dbPromise: Promise<IDBPDatabase<MatchMarkDB>> | null = null;

export function openSyncQueue(): Promise<IDBPDatabase<MatchMarkDB>> {
	if (!dbPromise) {
		dbPromise = openDB<MatchMarkDB>('matchmark', 1, {
			upgrade(db) {
				const store = db.createObjectStore('sync_queue', {
					keyPath: 'id',
					autoIncrement: true
				});
				store.createIndex('by_competition', 'competitionId');
			}
		});
	}
	return dbPromise;
}
