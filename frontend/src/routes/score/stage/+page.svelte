<script lang="ts">
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { session } from '$lib/stores/session';
	import { scoring } from '$lib/stores/scoring';
	import type { StageState } from '$lib/stores/scoring';
	import { apiFetch } from '$lib/api/client';
	import { enqueue, replayQueue, registerBackgroundSync } from '$lib/sync/queue';

	const RING_VALUES: Record<string, number> = { X: 10, '10': 10, '9': 9, '8': 8, '7': 7, '0': 0, M: 0 };
	const RING_KEYS = ['X', '10', '9', '8', '7', '0'];

	let sess = $derived(get(session));
	let sc = $state(get(scoring));
	scoring.subscribe((v) => { sc = v; });

	let stage = $derived(sc.stages[sc.currentStage - 1] as StageState | undefined);
	let readonly = $derived(!stage || stage.fullyLocked || sc.currentStage !== sc.currentStage);
	let isActive = $derived((s: StageState) => s.stageNumber === sc.currentStage);
	let syncError = $state(false);
	let adminOnly = $state(false);

	onMount(async () => {
		const s = get(session);
		if (!s.competitionId || !s.scoreLane) { goto('/score/select'); return; }
		await registerBackgroundSync().catch(() => undefined);
	});

	function addHit(ring: string) {
		if (adminOnly) return;
		if (!stage || stage.fullyLocked) return;
		if (stage.hits.length >= stage.expectedShots) return;
		scoring.update((sc) => {
			const stages = [...sc.stages];
			const idx = sc.currentStage - 1;
			const s = { ...stages[idx] };
			s.hits = [...s.hits, ring];
			s.stageScore = s.hits.reduce((a, h) => a + RING_VALUES[h], 0);
			s.xCount = s.hits.filter((h) => h === 'X').length;
			s.synced = false;
			stages[idx] = s;
			return { ...sc, stages };
		});
	}

	function undo() {
		if (adminOnly) return;
		if (!stage || stage.fullyLocked || stage.hits.length === 0) return;
		scoring.update((sc) => {
			const stages = [...sc.stages];
			const idx = sc.currentStage - 1;
			const s = { ...stages[idx] };
			s.hits = s.hits.slice(0, -1);
			s.stageScore = s.hits.reduce((a, h) => a + RING_VALUES[h], 0);
			s.xCount = s.hits.filter((h) => h === 'X').length;
			s.synced = false;
			stages[idx] = s;
			return { ...sc, stages };
		});
	}

	async function lockStage() {
		if (adminOnly) return;
		const s = get(session);
		const st = get(scoring);
		const cur = st.stages[st.currentStage - 1];
		if (!cur || !s.competitionId || !s.scoreLane) return;

		await submitHits(s.competitionId, s.scoreLane, cur);

		const res = await apiFetch(
			`/api/competitions/${s.competitionId}/lanes/${s.scoreLane}/stage/${cur.stageNumber}/lock`,
			{ method: 'POST' }
		);
		if (!res.ok) {
			if (res.status === 403) {
				const body = await res.json().catch(() => ({}));
				if (body?.error === 'admin scoring required') {
					adminOnly = true;
				}
			}
			return;
		}

		scoring.update((sc) => {
			const stages = [...sc.stages];
			const idx = sc.currentStage - 1;
			stages[idx] = { ...stages[idx], lockedByScorer: true, fullyLocked: true };
			return { ...sc, stages };
		});

		const totalStages = get(session).totalStages ?? sc.stages.length;
		if (st.currentStage >= totalStages) {
			goto('/score/summary');
		}
	}

	async function submitHits(compId: number, lane: number, st: StageState) {
		syncError = false;
		try {
			const res = await apiFetch(
				`/api/competitions/${compId}/lanes/${lane}/stage/${st.stageNumber}/hits`,
				{ method: 'POST', body: JSON.stringify({ hits: st.hits }) }
			);
			if (res.ok) {
				scoring.update((sc) => {
					const stages = [...sc.stages];
					const idx = st.stageNumber - 1;
					stages[idx] = { ...stages[idx], synced: true };
					return { ...sc, stages };
				});
			} else if (res.status === 403) {
				const body = await res.json().catch(() => ({}));
				if (body?.error === 'admin scoring required') {
					adminOnly = true;
					return;
				}
				throw new Error('submit failed');
			} else {
				throw new Error('submit failed');
			}
		} catch {
			await enqueue({ competitionId: compId, lane, stage: st.stageNumber, hits: st.hits });
			await registerBackgroundSync().catch(() => undefined);
			await replayQueue().catch(() => undefined);
			syncError = true;
		}
	}

	function prevStage() {
		if (sc.currentStage <= 1) return;
		scoring.update((sc) => ({ ...sc, currentStage: sc.currentStage - 1 }));
	}

	async function nextStage() {
		const s = get(session);
		const st = get(scoring);
		const cur = st.stages[st.currentStage - 1];
		if (!cur) return;
		if (!cur.fullyLocked && !cur.synced && cur.hits.length > 0 && s.competitionId && s.scoreLane) {
			await submitHits(s.competitionId, s.scoreLane, cur);
		}
		const totalStages = s.totalStages ?? st.stages.length;
		if (st.currentStage < totalStages) {
			scoring.update((sc) => ({ ...sc, currentStage: sc.currentStage + 1 }));
		}
	}

	function canNext(): boolean {
		const st = sc.stages[sc.currentStage - 1];
		if (!st) return false;
		if (st.fullyLocked) return true;
		return st.hits.length === 0;
	}

	function canLock(): boolean {
		const st = sc.stages[sc.currentStage - 1];
		if (!st) return false;
		return !st.fullyLocked && st.hits.length === st.expectedShots;
	}
</script>

<main class="stage-page">
	{#if syncError}
		<div class="sync-banner">Offline — hits saved locally, will sync when reconnected</div>
	{/if}

	{#if adminOnly}
		<div class="admin-banner">Admin scoring required — This competition has only one lane. Contact an admin to record scores.</div>
	{/if}

	<section class="progress-dots">
		{#each sc.stages as st (st.stageNumber)}
			<span
				class="dot"
				class:active={isActive(st)}
				class:locked={st.fullyLocked}
				aria-label="Stage {st.stageNumber}{st.fullyLocked ? ' locked' : ''}"
			>
				{#if st.fullyLocked}✓{:else}·{/if}
			</span>
		{/each}
	</section>

	{#if stage}
		<section class="stage-header">
			<p class="stage-title">
				Stage {stage.stageNumber} of {get(session).totalStages ?? sc.stages.length}
				· {stage.distance}
				· {stage.timeSec}s
			</p>
			<p class="stage-sub">{stage.position}</p>
		</section>

		<section class="score-display">
			<p>Shots: <strong>{stage.hits.length}</strong> / {stage.expectedShots}</p>
			<p>Score: <strong>{stage.stageScore}</strong> &nbsp; X: <strong>{stage.xCount}</strong></p>
		</section>

		<section class="keypad" class:readonly={stage.fullyLocked || adminOnly}>
			<div class="ring-grid">
				{#each RING_KEYS as ring (ring)}
					<button
						class="ring-btn"
						onclick={() => addHit(ring)}
						disabled={stage.fullyLocked || stage.hits.length >= stage.expectedShots}
					>{ring}</button>
				{/each}
			</div>
			<button
				class="miss-btn"
				onclick={() => addHit('M')}
				disabled={stage.fullyLocked || stage.hits.length >= stage.expectedShots}
			>M</button>
		</section>

		<section class="nav-row">
			<button class="nav-btn" onclick={undo} disabled={stage.fullyLocked || stage.hits.length === 0}>
				← Undo
			</button>
			<button class="nav-btn" onclick={prevStage} disabled={sc.currentStage <= 1}>
				← Prev
			</button>
			<button class="nav-btn" onclick={nextStage} disabled={!canNext()}>
				Next →
			</button>
		</section>

		<section class="lock-row">
			<button class="lock-btn" onclick={lockStage} disabled={!canLock()}>
				Lock Stage
			</button>
		</section>
	{:else}
		<p class="muted">Loading stage…</p>
	{/if}
</main>

<style>
	.stage-page {
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding: 0.75rem;
	}

	.sync-banner {
		background: rgba(234, 179, 8, 0.2);
		border: 1px solid rgba(234, 179, 8, 0.5);
		color: #fde047;
		padding: 0.5rem 0.875rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		text-align: center;
	}

	.admin-banner {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.5);
		color: #fca5a5;
		padding: 0.5rem 0.875rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		text-align: center;
	}

	.progress-dots {
		display: flex;
		gap: 0.375rem;
		flex-wrap: wrap;
		justify-content: center;
	}

	.dot {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		background: var(--surface);
		border: 2px solid var(--border);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
		color: var(--muted);
	}

	.dot.active {
		border-color: var(--accent);
		color: var(--accent);
		font-weight: 700;
	}

	.dot.locked {
		background: var(--accent);
		border-color: var(--accent);
		color: #fff;
	}

	.stage-header {
		text-align: center;
	}

	.stage-title {
		margin: 0;
		font-size: 0.9375rem;
		font-weight: 600;
	}

	.stage-sub {
		margin: 0.25rem 0 0;
		font-size: 0.8125rem;
		color: var(--muted);
	}

	.score-display {
		text-align: center;
		display: flex;
		gap: 1.5rem;
		justify-content: center;
	}

	.score-display p {
		margin: 0;
		font-size: 1rem;
	}

	.score-display strong {
		font-size: 1.5rem;
		color: var(--accent);
	}

	.keypad {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.keypad.readonly {
		opacity: 0.45;
		pointer-events: none;
	}

	.ring-grid {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 0.375rem;
	}

	.ring-btn {
		min-height: 72px;
		border: none;
		border-radius: 8px;
		background: var(--surface);
		color: var(--text);
		font-size: 1.125rem;
		font-weight: 700;
		cursor: pointer;
	}

	.ring-btn:active:not(:disabled) {
		background: var(--accent);
		color: #fff;
	}

	.ring-btn:disabled {
		opacity: 0.35;
		cursor: not-allowed;
	}

	.miss-btn {
		width: 100%;
		min-height: 72px;
		border: none;
		border-radius: 8px;
		background: #7f1d1d;
		color: #fca5a5;
		font-size: 1.25rem;
		font-weight: 700;
		cursor: pointer;
	}

	.miss-btn:active:not(:disabled) {
		background: #991b1b;
	}

	.miss-btn:disabled {
		opacity: 0.35;
		cursor: not-allowed;
	}

	.nav-row {
		display: grid;
		grid-template-columns: 1fr 1fr 1fr;
		gap: 0.375rem;
	}

	.nav-btn {
		min-height: 52px;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--surface);
		color: var(--text);
		font-size: 0.875rem;
		cursor: pointer;
	}

	.nav-btn:disabled {
		opacity: 0.35;
		cursor: not-allowed;
	}

	.lock-row {
		margin-top: auto;
	}

	.lock-btn {
		width: 100%;
		min-height: 56px;
		border: none;
		border-radius: 10px;
		background: var(--accent);
		color: #fff;
		font-size: 1.125rem;
		font-weight: 700;
		cursor: pointer;
	}

	.lock-btn:disabled {
		opacity: 0.35;
		cursor: not-allowed;
	}

	.muted {
		color: var(--muted);
		text-align: center;
	}
</style>
