<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';

	type SpectatorStage = {
		stage_number: number;
		score: number;
		x_count: number;
		locked: boolean;
	};

	type SpectatorLane = {
		lane_number: number;
		shooter_name: string;
		club: string | null;
		discipline: string;
		weapon_class: string;
		cumulative_score: number;
		cumulative_x_count: number;
		stages: SpectatorStage[];
	};

	type Snapshot = {
		competition_id: number;
		competition_name: string;
		lanes: SpectatorLane[];
	};

	let snapshot: Snapshot | null = $state(null);
	let notAvailable = $state(false);
	let totalStages = $state(0);

	let es: EventSource | null = null;
	let retryDelay = 1000;
	let retryTimer: ReturnType<typeof setTimeout> | null = null;
	let destroyed = false;

	function competitionID(): string {
		return get(page).params.id ?? '0';
	}

	function connect() {
		if (destroyed) return;
		es = new EventSource(`/api/spectator/competitions/${competitionID()}/live`);

		es.addEventListener('message', (e) => {
			const data: Snapshot = JSON.parse(e.data);
			snapshot = data;
			if (data.lanes.length > 0) {
				totalStages = Math.max(...data.lanes.map((l) => l.stages.length));
			}
			retryDelay = 1000;
			notAvailable = false;
		});

		es.addEventListener('error', (e: Event) => {
			const me = e as MessageEvent;
			if (me.data) {
				notAvailable = true;
			}
			es?.close();
			es = null;
			if (!destroyed && !notAvailable) {
				retryTimer = setTimeout(() => {
					retryDelay = Math.min(retryDelay * 2, 30000);
					connect();
				}, retryDelay);
			}
		});

		es.onerror = () => {
			es?.close();
			es = null;
			if (!destroyed && !notAvailable) {
				retryTimer = setTimeout(() => {
					retryDelay = Math.min(retryDelay * 2, 30000);
					connect();
				}, retryDelay);
			}
		};
	}

	onMount(() => {
		connect();
	});

	onDestroy(() => {
		destroyed = true;
		if (retryTimer !== null) clearTimeout(retryTimer);
		es?.close();
	});

	function stageScore(lane: SpectatorLane, stageNumber: number): string {
		const s = lane.stages.find((st) => st.stage_number === stageNumber);
		if (!s) return '';
		if (!s.locked) return '—';
		return String(s.score);
	}

	let stageNumbers: number[] = $derived(
		totalStages > 0 ? Array.from({ length: totalStages }, (_, i) => i + 1) : []
	);
</script>

<div class="page">
	{#if notAvailable}
		<div class="unavailable">
			<h2>Not Available</h2>
			<p>Competition not found or not available for public viewing.</p>
		</div>
	{:else if !snapshot}
		<p class="muted">Connecting to live scoreboard...</p>
	{:else}
		<div class="header">
			<h1>{snapshot.competition_name}</h1>
			<span class="live-badge">LIVE</span>
		</div>

		{#if snapshot.lanes.length === 0}
			<p class="muted">No results yet.</p>
		{:else}
			<div class="table-wrap">
				<table>
					<thead>
						<tr>
							<th>Rank</th>
							<th>Lane</th>
							<th>Name</th>
							<th>Club</th>
							<th>Discipline</th>
							<th>Weapon</th>
							{#each stageNumbers as n}
								<th>St.{n}</th>
							{/each}
							<th>Total</th>
							<th>X</th>
						</tr>
					</thead>
					<tbody>
						{#each snapshot.lanes as lane, i (lane.lane_number)}
							<tr>
								<td class="rank">{i + 1}</td>
								<td>{lane.lane_number}</td>
								<td class="name">{lane.shooter_name}</td>
								<td class="muted">{lane.club ?? ''}</td>
								<td>{lane.discipline}</td>
								<td>{lane.weapon_class}</td>
								{#each stageNumbers as n}
									<td class="stage-cell">{stageScore(lane, n)}</td>
								{/each}
								<td class="total">{lane.cumulative_score}</td>
								<td class="x-count">{lane.cumulative_x_count}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</div>

<style>
	.page {
		padding: 1.5rem 1rem;
		max-width: 1200px;
		margin: 0 auto;
	}

	.header {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	h1 {
		margin: 0;
		color: var(--text);
	}

	.live-badge {
		background: #e74c3c;
		color: #fff;
		font-size: 0.75rem;
		font-weight: 700;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		letter-spacing: 0.05em;
	}

	.muted {
		color: var(--muted);
	}

	.unavailable {
		text-align: center;
		margin-top: 4rem;
		color: var(--muted);
	}

	.unavailable h2 {
		color: var(--text);
	}

	.table-wrap {
		overflow-x: auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.9rem;
	}

	th {
		text-align: left;
		padding: 0.5rem 0.75rem;
		border-bottom: 2px solid var(--border);
		color: var(--muted);
		white-space: nowrap;
	}

	td {
		padding: 0.5rem 0.75rem;
		border-bottom: 1px solid var(--border);
		color: var(--text);
		white-space: nowrap;
	}

	.rank {
		font-weight: 700;
		color: var(--accent);
	}

	.name {
		font-weight: 600;
	}

	.total {
		font-weight: 700;
	}

	.x-count {
		color: var(--accent);
	}

	.stage-cell {
		color: var(--muted);
		text-align: right;
	}

	tbody tr:hover {
		background: var(--surface);
	}
</style>
