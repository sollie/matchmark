<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { apiFetch } from '$lib/api/client';
	import { session } from '$lib/stores/session';
	import { scoring } from '$lib/stores/scoring';
	import type { StageState } from '$lib/stores/scoring';

	interface Competition {
		id: number;
		name: string;
		date: string;
		status: string;
		discipline: string;
	}

	interface Shooter {
		id: number;
		name: string;
		club: string;
		lane: number;
		discipline: string;
		weapon_class: string;
	}

	interface Assignment {
		shooter_id: number;
		lane_number: number;
		shooter_name: string;
		competition_id: number;
	}

	interface StageDetail {
		stage_number: number;
		expected_shots: number;
		distance: string;
		time_sec: number;
		position: string;
		hits: string[];
		locked_by_scorer: boolean;
		fully_locked: boolean;
		stage_score: number;
		stage_x_count: number;
		total_stages: number;
	}

	let competitions = $state<Competition[]>([]);
	let selectedComp = $state<Competition | null>(null);
	let shooters = $state<Shooter[]>([]);
	let loadError = $state('');
	let claiming = $state<number | null>(null);
	let claimError = $state('');

	function scoreLane(own: number, totalLanes: number): number {
		return ((own - 2 + totalLanes) % totalLanes) + 1;
	}

	async function loadStages(compId: number, lane: number, totalStages: number): Promise<StageState[]> {
		const stages: StageState[] = [];
		for (let s = 1; s <= totalStages; s++) {
			try {
				const res = await apiFetch(`/api/competitions/${compId}/lanes/${lane}/stage/${s}`);
				if (res.ok) {
					const d: StageDetail = await res.json();
					stages.push({
						stageNumber: d.stage_number,
						expectedShots: d.expected_shots,
						distance: d.distance,
						timeSec: d.time_sec,
						position: d.position,
						hits: d.hits ?? [],
						lockedByScorer: d.locked_by_scorer,
						fullyLocked: d.fully_locked,
						stageScore: d.stage_score,
						xCount: d.stage_x_count,
						synced: true
					});
				}
			} catch {
				// keep going; stage will appear blank
			}
		}
		return stages;
	}

	onMount(async () => {
		try {
			const res = await apiFetch('/api/competitions');
			if (!res.ok) { loadError = 'Failed to load competitions'; return; }
			const all: Competition[] = await res.json();
			competitions = all.filter((c) => c.status === 'active');
		} catch {
			loadError = 'Network error loading competitions';
			return;
		}

		if (competitions.length === 1) {
			await selectCompetition(competitions[0]);
		}
	});

	async function selectCompetition(comp: Competition) {
		selectedComp = comp;
		claimError = '';
		loadError = '';

		const assignRes = await apiFetch(`/api/competitions/${comp.id}/assign`);
		if (assignRes.ok) {
			const a: Assignment = await assignRes.json();
			const sl = scoreLane(a.lane_number, shooters.length || a.lane_number);
			session.set({
				competitionId: comp.id,
				competitionName: comp.name,
				ownLane: a.lane_number,
				scoreLane: sl,
				totalStages: null
			});

			const stageRes = await apiFetch(`/api/competitions/${comp.id}/lanes/${sl}/stage/1`);
			if (stageRes.ok) {
				await navigateToStage(comp.id, sl);
			}
			return;
		}

		try {
			const res = await apiFetch(`/api/competitions/${comp.id}/shooters`);
			if (!res.ok) { loadError = 'Failed to load shooters'; return; }
			shooters = await res.json();
		} catch {
			loadError = 'Network error loading shooters';
		}
	}

	async function claimShooter(shooter: Shooter) {
		if (!selectedComp) return;
		claiming = shooter.id;
		claimError = '';
		try {
			const res = await apiFetch(`/api/competitions/${selectedComp.id}/assign`, {
				method: 'POST',
				body: JSON.stringify({ shooter_id: shooter.id })
			});
			if (!res.ok) {
				const d = await res.json().catch(() => ({}));
				claimError = d.error ?? 'Could not claim shooter';
				return;
			}
			const a: Assignment = await res.json();
			const sl = scoreLane(a.lane_number, shooters.length);
			session.set({
				competitionId: selectedComp.id,
				competitionName: selectedComp.name,
				ownLane: a.lane_number,
				scoreLane: sl,
				totalStages: null
			});
			await navigateToStage(selectedComp.id, sl);
		} catch {
			claimError = 'Network error claiming shooter';
		} finally {
			claiming = null;
		}
	}

	async function navigateToStage(compId: number, lane: number) {
		const stageRes = await apiFetch(`/api/competitions/${compId}/lanes/${lane}/stage/1`);
		if (!stageRes.ok) { goto('/score/stage'); return; }
		const first: StageDetail = await stageRes.json();

		const totalStages = first.total_stages;

		session.update((s) => ({ ...s, totalStages }));
		const stages = await loadStages(compId, lane, totalStages);

		let currentStage = 1;
		for (const st of stages) {
			if (st.fullyLocked) currentStage = st.stageNumber + 1;
			else break;
		}
		currentStage = Math.min(currentStage, totalStages);

		scoring.set({ currentStage, stages });
		goto('/score/stage');
	}
</script>

<main class="select-page">
	<header class="page-header">
		<h1>Select your lane</h1>
	</header>

	{#if loadError}
		<p class="error">{loadError}</p>
	{/if}

	{#if !selectedComp}
		<section>
			<h2>Active competitions</h2>
			{#if competitions.length === 0 && !loadError}
				<p class="muted">No active competitions.</p>
			{/if}
			<ul class="card-list">
				{#each competitions as comp (comp.id)}
					<li>
						<button class="comp-card" onclick={() => selectCompetition(comp)}>
							<span class="comp-name">{comp.name}</span>
							<span class="comp-meta">{comp.date} · {comp.discipline}</span>
						</button>
					</li>
				{/each}
			</ul>
		</section>
	{:else}
		<section>
			<button class="back-btn" onclick={() => { selectedComp = null; shooters = []; }}>← Back</button>
			<h2>{selectedComp.name}</h2>
			<p class="muted">Tap your name to claim your lane.</p>
			{#if claimError}
				<p class="error">{claimError}</p>
			{/if}
			<ul class="card-list">
				{#each shooters as shooter (shooter.id)}
					<li>
						<button
							class="shooter-card"
							onclick={() => claimShooter(shooter)}
							disabled={claiming !== null}
						>
							<span class="shooter-name">{shooter.name}</span>
							<span class="shooter-meta">Lane {shooter.lane} · {shooter.club}</span>
						</button>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</main>

<style>
	.select-page {
		min-height: 100dvh;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.page-header h1 {
		margin: 0;
		font-size: 1.5rem;
		color: var(--accent);
	}

	h2 {
		margin: 0 0 0.5rem;
		font-size: 1.125rem;
	}

	.error {
		padding: 0.625rem 0.875rem;
		background: rgba(220, 38, 38, 0.15);
		border: 1px solid rgba(220, 38, 38, 0.4);
		border-radius: 6px;
		color: #f87171;
		font-size: 0.875rem;
		margin: 0;
	}

	.muted {
		color: var(--muted);
		font-size: 0.875rem;
		margin: 0;
	}

	.card-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.comp-card,
	.shooter-card {
		width: 100%;
		padding: 1rem;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		cursor: pointer;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		text-align: left;
		min-height: 72px;
	}

	.comp-card:active,
	.shooter-card:active {
		background: var(--surface-active);
	}

	.comp-card:disabled,
	.shooter-card:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.comp-name,
	.shooter-name {
		font-size: 1rem;
		font-weight: 600;
		color: var(--text);
	}

	.comp-meta,
	.shooter-meta {
		font-size: 0.8125rem;
		color: var(--muted);
	}

	.back-btn {
		background: none;
		border: none;
		color: var(--accent);
		font-size: 0.875rem;
		cursor: pointer;
		padding: 0;
		margin-bottom: 0.5rem;
	}
</style>
