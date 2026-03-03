import { writable } from 'svelte/store';

export interface StageState {
	stageNumber: number;
	expectedShots: number;
	distance: string;
	timeSec: number;
	position: string;
	hits: string[];
	lockedByScorer: boolean;
	fullyLocked: boolean;
	stageScore: number;
	xCount: number;
	synced: boolean;
}

export interface ScoringState {
	currentStage: number;
	stages: StageState[];
}

export const scoring = writable<ScoringState>({
	currentStage: 1,
	stages: []
});
