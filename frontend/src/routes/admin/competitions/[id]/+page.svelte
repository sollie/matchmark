<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { get } from 'svelte/store';
	import { apiFetch } from '$lib/api/client';

	const RING_KEYS = ['X', '10', '9', '8', '7', '0'];
	const RING_VALUES: Record<string, number> = { X: 10, '10': 10, '9': 9, '8': 8, '7': 7, '0': 0, M: 0 };
	const WEAPON_CLASSES: Record<string, string[]> = {
		'1500': ['Revolver 1500', 'Pistol 1500', 'Revolver Optical Sight 1500', 'Pistol Optical Sight 1500'],
		'60': ['Distinguished Revolver', 'Distinguished Pistol', 'Open Match'],
		'48': [
			'Standard Revolver 4.25"',
			'Standard Revolver 2.75"',
			'Standard Revolver 2.75" 5-Shot',
			'Standard Semi-Automatic Pistol 5.5" Production',
			'Standard Semi-Automatic Pistol 5.5"',
			'Standard Semi-Automatic Pistol 3.7"'
		]
	};

	interface MonitorLane {
		lane_number: number;
		shooter_name: string;
		discipline: string;
		weapon_class: string;
		current_stage: number;
		stage_fully_locked: boolean;
		scorer_locked: boolean;
		cumulative_score: number;
		x_count: number;
	}

	interface MonitorEvent {
		competition_id: number;
		competition_name: string;
		status: string;
		lanes: MonitorLane[];
	}

	interface StageDetail {
		stage_number: number;
		expected_shots: number;
		distance: string;
		time_seconds: number;
		position: string;
		hits: { ring: string; points: number }[];
		locked_by_scorer: boolean;
		fully_locked: boolean;
		stage_score: number;
		stage_x_count: number;
	}

	interface Shooter {
		id: number;
		name: string;
		wa_id: string;
		club: string;
		lane_number: number;
		discipline: string;
		weapon_class: string;
	}

	interface Competition {
		id: number;
		name: string;
		date: string;
		lanes_total: number;
		status: 'draft' | 'active' | 'finished';
		spectator_enabled: boolean;
	}

	const compId = parseInt(get(page).params.id ?? '0', 10);

	let comp = $state<Competition | null>(null);
	let monitor = $state<MonitorEvent | null>(null);
	let loadError = $state('');
	let sseError = $state('');

	let selectedLane = $state<MonitorLane | null>(null);
	let panelStages = $state<StageDetail[]>([]);
	let panelLoading = $state(false);

	let editStage = $state<StageDetail | null>(null);
	let editLane = $state(0);
	let editHits = $state<string[]>([]);
	let editSaving = $state(false);
	let editError = $state('');

	let addShooterOpen = $state(false);
	let addName = $state('');
	let addWaId = $state('');
	let addClub = $state('');
	let addLane = $state('');
	let addDiscipline = $state('1500');
	let addWeaponClass = $state(WEAPON_CLASSES['1500'][0]);
	let addError = $state('');
	let addSaving = $state(false);

	let sse: EventSource | null = null;
	let sseBackoff = 1000;

	async function connectSSE() {
		const ticketRes = await apiFetch('/api/admin/sse-ticket', {
			method: 'POST',
			body: JSON.stringify({ competition_id: compId })
		});
		if (!ticketRes.ok) {
			sseError = 'Failed to obtain monitor ticket';
			return;
		}
		const { ticket } = await ticketRes.json() as { ticket: string };
		sse = new EventSource(`/api/admin/monitor?ticket=${encodeURIComponent(ticket)}&competition_id=${compId}`);
		sse.onmessage = (e) => {
			try {
				monitor = JSON.parse(e.data) as MonitorEvent;
				sseError = '';
				sseBackoff = 1000;
			} catch { /* ignore parse errors */ }
		};
		sse.onerror = () => {
			sse?.close();
			sseError = 'Monitor disconnected — reconnecting…';
			setTimeout(connectSSE, sseBackoff);
			sseBackoff = Math.min(sseBackoff * 2, 30000);
		};
	}

	async function loadComp() {
		const res = await apiFetch(`/api/competitions/${compId}`);
		if (!res.ok) { loadError = 'Competition not found'; return; }
		comp = await res.json();
	}

	async function openPanel(lane: MonitorLane) {
		selectedLane = lane;
		panelLoading = true;
		panelStages = [];
		const totalStages = lane.discipline === '1500' ? 10 : 4;
		const results: StageDetail[] = [];
		for (let s = 1; s <= totalStages; s++) {
			const res = await apiFetch(`/api/competitions/${compId}/lanes/${lane.lane_number}/stage/${s}`);
			if (res.ok) results.push(await res.json());
		}
		panelStages = results;
		panelLoading = false;
	}

	async function unlockStage(stageNumber: number) {
		if (!selectedLane) return;
		await apiFetch(
			`/api/competitions/${compId}/lanes/${selectedLane.lane_number}/stage/${stageNumber}/unlock`,
			{ method: 'POST' }
		);
		await openPanel(selectedLane);
	}

	async function unlockAll() {
		if (!selectedLane) return;
		for (const st of panelStages) {
			if (st.fully_locked) {
				await apiFetch(
					`/api/competitions/${compId}/lanes/${selectedLane.lane_number}/stage/${st.stage_number}/unlock`,
					{ method: 'POST' }
				);
			}
		}
		await openPanel(selectedLane);
	}

	function openEditHits(st: StageDetail, lane: number) {
		editStage = st;
		editLane = lane;
		editHits = st.hits.map((h) => h.ring);
		editError = '';
	}

	function addEditHit(ring: string) {
		if (!editStage) return;
		if (editHits.length >= editStage.expected_shots) return;
		editHits = [...editHits, ring];
	}

	function undoEditHit() {
		editHits = editHits.slice(0, -1);
	}

	async function submitEditHits() {
		if (!editStage || !selectedLane) return;
		editError = '';
		editSaving = true;
		try {
			const res = await apiFetch(
				`/api/competitions/${compId}/lanes/${editLane}/stage/${editStage.stage_number}/hits`,
				{ method: 'POST', body: JSON.stringify({ hits: editHits }) }
			);
			if (!res.ok) {
				const d = await res.json().catch(() => ({}));
				editError = d.error ?? 'Failed to save hits';
				return;
			}
			editStage = null;
			await openPanel(selectedLane);
		} catch {
			editError = 'Network error';
		} finally {
			editSaving = false;
		}
	}

	async function setStatus(status: Competition['status']) {
		if (!comp) return;
		const res = await apiFetch(`/api/competitions/${compId}`, {
			method: 'PUT',
			body: JSON.stringify({ status })
		});
		if (res.ok) comp = await res.json();
	}

	function openAddShooter() {
		addName = '';
		addWaId = '';
		addClub = '';
		addLane = '';
		addDiscipline = '1500';
		addWeaponClass = WEAPON_CLASSES['1500'][0];
		addError = '';
		addShooterOpen = true;
	}

	async function submitAddShooter() {
		addError = '';
		addSaving = true;
		try {
			const res = await apiFetch(`/api/competitions/${compId}/shooters`, {
				method: 'POST',
				body: JSON.stringify({
					name: addName,
					wa_id: addWaId,
					club: addClub || undefined,
					lane_number: parseInt(addLane, 10),
					discipline: addDiscipline,
					weapon_class: addWeaponClass
				})
			});
			if (!res.ok) {
				const d = await res.json().catch(() => ({}));
				addError = d.error ?? 'Failed to add shooter';
				return;
			}
			addShooterOpen = false;
		} catch {
			addError = 'Network error';
		} finally {
			addSaving = false;
		}
	}

	let editHitScore = $derived(editHits.reduce((a, h) => a + RING_VALUES[h], 0));
	let editHitX = $derived(editHits.filter((h) => h === 'X').length);

	onMount(() => {
		loadComp();
		connectSSE();
	});

	onDestroy(() => {
		sse?.close();
	});
</script>

<main class="admin-page">
	<header class="page-header">
		<div class="breadcrumb">
			<a href="/admin">Admin</a>
			<span>/</span>
			<a href="/admin/competitions">Competitions</a>
			<span>/ {comp?.name ?? '…'}</span>
		</div>
		<div class="header-actions">
			{#if comp}
				{#if comp.status === 'draft'}
					<button class="btn-sm" onclick={() => setStatus('active')}>Activate</button>
				{:else if comp.status === 'active'}
					<button class="btn-sm" onclick={() => setStatus('finished')}>Finish</button>
				{/if}
				<a href="/admin/competitions/{compId}/export" class="btn-sm">Export</a>
				<button class="btn-primary" onclick={openAddShooter}>+ Shooter</button>
			{/if}
		</div>
	</header>

	{#if loadError}
		<p class="error">{loadError}</p>
	{/if}

	{#if comp}
		<div class="comp-meta">
			<span>{comp.date}</span>
			<span>·</span>
			<span>{comp.lanes_total} lanes</span>
			<span>·</span>
			<span class="badge status-{comp.status}">{comp.status}</span>
			{#if comp.spectator_enabled}
				<span class="badge spectator">Live</span>
			{/if}
		</div>
	{/if}

	{#if sseError}
		<p class="warn">{sseError}</p>
	{/if}

	{#if monitor}
		<div class="table-wrap">
			<table>
				<thead>
					<tr>
						<th>Lane</th>
						<th>Shooter</th>
						<th>Discipline</th>
						<th>Stage</th>
						<th>Score</th>
						<th>X</th>
						<th>Locked</th>
					</tr>
				</thead>
				<tbody>
					{#each monitor.lanes as lane (lane.lane_number)}
						<tr class="clickable" onclick={() => openPanel(lane)}>
							<td>{lane.lane_number}</td>
							<td>{lane.shooter_name}</td>
							<td>{lane.discipline}</td>
							<td>{lane.current_stage}</td>
							<td><strong>{lane.cumulative_score}</strong></td>
							<td>{lane.x_count}</td>
							<td>{lane.stage_fully_locked ? '✓' : '—'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else if !loadError}
		<p class="muted">Waiting for monitor data…</p>
	{/if}
</main>

{#if selectedLane}
	<div class="panel-overlay" onclick={() => { selectedLane = null; }} onkeydown={(e) => { if (e.key === 'Escape') selectedLane = null; }} role="presentation" tabindex="-1">
		<div class="panel" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Lane {selectedLane.lane_number} details" tabindex="0">
			<div class="panel-header">
				<div>
					<h2>Lane {selectedLane.lane_number} — {selectedLane.shooter_name}</h2>
					<p class="muted">{selectedLane.discipline} · {selectedLane.weapon_class}</p>
				</div>
				<button class="close-btn" onclick={() => { selectedLane = null; }}>✕</button>
			</div>

			{#if panelLoading}
				<p class="muted">Loading…</p>
			{:else}
				<div class="stage-list">
					{#each panelStages as st (st.stage_number)}
						<div class="stage-row">
							<div class="stage-info">
								<span class="stage-num">Stage {st.stage_number}</span>
								<span class="stage-score">{st.stage_score} pts · {st.stage_x_count}X</span>
								<span class="stage-meta">{st.distance} · {st.position}</span>
								<span class="stage-hits">{st.hits.map((h) => h.ring).join(' ')}</span>
							</div>
							<div class="stage-actions">
								{#if st.fully_locked}
									<span class="locked-badge">Locked</span>
								{/if}
								<button class="btn-sm" onclick={() => openEditHits(st, selectedLane!.lane_number)}>Edit hits</button>
								{#if st.fully_locked}
									<button class="btn-sm" onclick={() => unlockStage(st.stage_number)}>Unlock</button>
								{/if}
							</div>
						</div>
					{/each}
				</div>

				<div class="panel-footer">
					<button class="btn-secondary" onclick={unlockAll}>Unlock all stages</button>
				</div>
			{/if}
		</div>
	</div>
{/if}

{#if editStage}
	<div class="overlay" role="dialog" aria-modal="true" aria-label="Edit hits">
		<div class="modal modal-wide">
			<div class="modal-header">
				<h2>Edit hits — Stage {editStage.stage_number} (Lane {editLane})</h2>
				<p class="muted">{editStage.distance} · {editStage.expected_shots} shots expected</p>
			</div>

			<div class="score-display">
				<span>Shots: <strong>{editHits.length}</strong> / {editStage.expected_shots}</span>
				<span>Score: <strong>{editHitScore}</strong></span>
				<span>X: <strong>{editHitX}</strong></span>
			</div>

			<div class="hits-display">
				{#each editHits as h, i (i)}
					<span class="hit-chip" class:x-hit={h === 'X'} class:miss={h === 'M'}>{h}</span>
				{/each}
			</div>

			<div class="keypad">
				<div class="ring-grid">
					{#each RING_KEYS as ring (ring)}
						<button
							class="ring-btn"
							onclick={() => addEditHit(ring)}
							disabled={editSaving || editHits.length >= editStage.expected_shots}
						>{ring}</button>
					{/each}
				</div>
				<button
					class="miss-btn"
					onclick={() => addEditHit('M')}
					disabled={editSaving || editHits.length >= editStage.expected_shots}
				>M</button>
			</div>

			<div class="undo-row">
				<button class="btn-secondary" onclick={undoEditHit} disabled={editSaving || editHits.length === 0}>← Undo</button>
			</div>

			{#if editError}
				<p class="error">{editError}</p>
			{/if}

			<div class="modal-actions">
				<button class="btn-secondary" onclick={() => editStage = null} disabled={editSaving}>Cancel</button>
				<button
					class="btn-primary"
					onclick={submitEditHits}
					disabled={editSaving || editHits.length !== editStage.expected_shots}
				>
					{editSaving ? 'Saving…' : 'Save hits'}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if addShooterOpen}
	<div class="overlay" role="dialog" aria-modal="true" aria-label="Add shooter">
		<div class="modal">
			<h2>Add shooter</h2>
			<form onsubmit={(e) => { e.preventDefault(); submitAddShooter(); }}>
				<label>
					Name
					<input type="text" bind:value={addName} required disabled={addSaving} />
				</label>
				<label>
					WA-ID
					<input type="text" bind:value={addWaId} required disabled={addSaving} />
				</label>
				<label>
					Club
					<input type="text" bind:value={addClub} disabled={addSaving} />
				</label>
				<label>
					Lane number
					<input type="number" min="1" bind:value={addLane} required disabled={addSaving} />
				</label>
				<label>
					Discipline
					<select bind:value={addDiscipline} onchange={() => { addWeaponClass = WEAPON_CLASSES[addDiscipline][0]; }} disabled={addSaving}>
						<option value="1500">1500</option>
						<option value="60">60-shot</option>
						<option value="48">48-shot</option>
					</select>
				</label>
				<label>
					Weapon class
					<select bind:value={addWeaponClass} disabled={addSaving}>
						{#each WEAPON_CLASSES[addDiscipline] as wc (wc)}
							<option value={wc}>{wc}</option>
						{/each}
					</select>
				</label>
				{#if addError}
					<p class="error">{addError}</p>
				{/if}
				<div class="modal-actions">
					<button type="button" class="btn-secondary" onclick={() => addShooterOpen = false} disabled={addSaving}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={addSaving || !addName || !addWaId || !addLane}>
						{addSaving ? 'Adding…' : 'Add shooter'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.admin-page {
		min-height: 100dvh;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.9375rem;
	}

	.breadcrumb a {
		color: var(--accent);
		text-decoration: none;
	}

	.breadcrumb span {
		color: var(--muted);
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.comp-meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: var(--muted);
		flex-wrap: wrap;
	}

	.badge {
		display: inline-block;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		border: 1px solid var(--border);
		color: var(--muted);
		background: var(--surface);
	}

	.badge.status-active {
		background: rgba(34, 197, 94, 0.15);
		border-color: rgba(34, 197, 94, 0.4);
		color: #4ade80;
	}

	.badge.status-finished {
		background: rgba(240, 192, 64, 0.1);
		border-color: rgba(240, 192, 64, 0.3);
		color: var(--accent);
	}

	.badge.spectator {
		background: rgba(99, 102, 241, 0.15);
		border-color: rgba(99, 102, 241, 0.4);
		color: #a5b4fc;
	}

	.error {
		margin: 0;
		padding: 0.625rem 0.875rem;
		background: rgba(220, 38, 38, 0.15);
		border: 1px solid rgba(220, 38, 38, 0.4);
		border-radius: 6px;
		color: #f87171;
		font-size: 0.875rem;
	}

	.warn {
		margin: 0;
		padding: 0.625rem 0.875rem;
		background: rgba(234, 179, 8, 0.15);
		border: 1px solid rgba(234, 179, 8, 0.4);
		border-radius: 6px;
		color: #fde047;
		font-size: 0.875rem;
	}

	.muted {
		color: var(--muted);
		font-size: 0.875rem;
		margin: 0;
	}

	.table-wrap {
		overflow-x: auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.9375rem;
	}

	th {
		text-align: left;
		padding: 0.625rem 0.75rem;
		border-bottom: 1px solid var(--border);
		color: var(--muted);
		font-weight: 600;
		font-size: 0.8125rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		white-space: nowrap;
	}

	td {
		padding: 0.75rem;
		border-bottom: 1px solid var(--border);
		vertical-align: middle;
	}

	tr.clickable {
		cursor: pointer;
	}

	tr.clickable:hover td {
		background: var(--surface);
	}

	.btn-primary {
		padding: 0.625rem 1.25rem;
		border: none;
		border-radius: 8px;
		background: var(--accent);
		color: #1a1a2e;
		font-size: 0.9375rem;
		font-weight: 700;
		cursor: pointer;
		min-height: 44px;
	}

	.btn-primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-secondary {
		padding: 0.625rem 1.25rem;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--surface);
		color: var(--text);
		font-size: 0.9375rem;
		cursor: pointer;
		min-height: 44px;
	}

	.btn-secondary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-sm {
		padding: 0.375rem 0.75rem;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--surface);
		color: var(--text);
		font-size: 0.8125rem;
		cursor: pointer;
		text-decoration: none;
		white-space: nowrap;
	}

	.btn-sm:active {
		background: var(--surface-active);
	}

	.panel-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		z-index: 50;
		display: flex;
		justify-content: flex-end;
	}

	.panel {
		width: min(480px, 100vw);
		height: 100%;
		background: var(--surface);
		border-left: 1px solid var(--border);
		display: flex;
		flex-direction: column;
		overflow-y: auto;
	}

	.panel-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		padding: 1.25rem;
		border-bottom: 1px solid var(--border);
		gap: 0.75rem;
	}

	.panel-header h2 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--muted);
		font-size: 1.125rem;
		cursor: pointer;
		flex-shrink: 0;
	}

	.stage-list {
		display: flex;
		flex-direction: column;
		flex: 1;
	}

	.stage-row {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 0.75rem;
		padding: 1rem 1.25rem;
		border-bottom: 1px solid var(--border);
	}

	.stage-info {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		flex: 1;
		min-width: 0;
	}

	.stage-num {
		font-weight: 600;
		font-size: 0.9375rem;
	}

	.stage-score {
		font-size: 0.875rem;
		color: var(--accent);
	}

	.stage-meta {
		font-size: 0.8125rem;
		color: var(--muted);
	}

	.stage-hits {
		font-size: 0.8125rem;
		color: var(--muted);
		word-break: break-all;
	}

	.stage-actions {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		align-items: flex-end;
		flex-shrink: 0;
	}

	.locked-badge {
		font-size: 0.75rem;
		color: #4ade80;
		font-weight: 600;
	}

	.panel-footer {
		padding: 1.25rem;
		border-top: 1px solid var(--border);
	}

	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
		z-index: 100;
	}

	.modal {
		width: 100%;
		max-width: 420px;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 12px;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.modal-wide {
		max-width: 540px;
	}

	.modal-header h2 {
		margin: 0 0 0.25rem;
		font-size: 1.05rem;
	}

	.modal h2 {
		margin: 0;
		font-size: 1.125rem;
	}

	.modal form {
		display: flex;
		flex-direction: column;
		gap: 0.875rem;
	}

	label {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		font-size: 0.875rem;
		color: var(--text);
	}

	input, select {
		padding: 0.625rem 0.875rem;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--bg);
		color: var(--text);
		font-size: 1rem;
	}

	input:focus, select:focus {
		outline: 2px solid var(--accent);
		outline-offset: 2px;
	}

	.modal-actions {
		display: flex;
		gap: 0.75rem;
		justify-content: flex-end;
	}

	.score-display {
		display: flex;
		gap: 1.5rem;
		font-size: 0.9375rem;
	}

	.score-display strong {
		color: var(--accent);
	}

	.hits-display {
		display: flex;
		flex-wrap: wrap;
		gap: 0.375rem;
		min-height: 2.5rem;
		padding: 0.5rem;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: 8px;
	}

	.hit-chip {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: 6px;
		background: var(--surface);
		font-size: 0.875rem;
		font-weight: 600;
	}

	.hit-chip.x-hit {
		background: rgba(240, 192, 64, 0.2);
		color: var(--accent);
	}

	.hit-chip.miss {
		background: rgba(220, 38, 38, 0.2);
		color: #f87171;
	}

	.keypad {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.ring-grid {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 0.375rem;
	}

	.ring-btn {
		min-height: 56px;
		border: none;
		border-radius: 8px;
		background: var(--bg);
		color: var(--text);
		font-size: 1rem;
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
		min-height: 56px;
		border: none;
		border-radius: 8px;
		background: #7f1d1d;
		color: #fca5a5;
		font-size: 1.125rem;
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

	.undo-row {
		display: flex;
	}
</style>
