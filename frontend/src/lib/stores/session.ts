import { writable } from 'svelte/store';

export interface SessionState {
	competitionId: number | null;
	competitionName: string | null;
	ownLane: number | null;
	scoreLane: number | null;
	totalStages: number | null;
}

export const session = writable<SessionState>({
	competitionId: null,
	competitionName: null,
	ownLane: null,
	scoreLane: null,
	totalStages: null
});
