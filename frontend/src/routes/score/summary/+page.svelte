<script lang="ts">
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session';
	import { scoring } from '$lib/stores/scoring';

	let sc = $state(get(scoring));
	scoring.subscribe((v) => { sc = v; });

	let sess = $derived(get(session));

	let totalScore = $derived(sc.stages.reduce((a, s) => a + s.stageScore, 0));
	let totalX = $derived(sc.stages.reduce((a, s) => a + s.xCount, 0));
</script>

<main class="summary-page">
	<header class="summary-header">
		<h1>Score Summary</h1>
		<p class="comp-name">{get(session).competitionName ?? ''}</p>
	</header>

	<table class="summary-table">
		<thead>
			<tr>
				<th>Stage</th>
				<th>Dist</th>
				<th>Shots</th>
				<th>Score</th>
				<th>X</th>
				<th>Hits</th>
			</tr>
		</thead>
		<tbody>
			{#each sc.stages as st (st.stageNumber)}
				<tr class:locked={st.fullyLocked}>
					<td>{st.stageNumber}</td>
					<td>{st.distance}</td>
					<td>{st.hits.length}/{st.expectedShots}</td>
					<td><strong>{st.stageScore}</strong></td>
					<td>{st.xCount}</td>
					<td class="hits-cell">{st.hits.join(' ')}</td>
				</tr>
			{/each}
		</tbody>
		<tfoot>
			<tr class="total-row">
				<td colspan="3">Total</td>
				<td><strong>{totalScore}</strong></td>
				<td>{totalX}</td>
				<td></td>
			</tr>
		</tfoot>
	</table>

	<footer class="summary-footer">
		<button class="back-btn" onclick={() => goto('/score/select')}>
			New Competition
		</button>
	</footer>
</main>

<style>
	.summary-page {
		min-height: 100dvh;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.summary-header h1 {
		margin: 0;
		font-size: 1.5rem;
		color: var(--accent);
	}

	.comp-name {
		margin: 0.25rem 0 0;
		color: var(--muted);
		font-size: 0.875rem;
	}

	.summary-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}

	.summary-table th {
		text-align: left;
		padding: 0.5rem 0.5rem;
		border-bottom: 1px solid var(--border);
		color: var(--muted);
		font-weight: 600;
		font-size: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.summary-table td {
		padding: 0.625rem 0.5rem;
		border-bottom: 1px solid var(--border);
		color: var(--text);
		vertical-align: top;
	}

	.summary-table tr.locked td {
		color: var(--text);
	}

	.hits-cell {
		font-size: 0.75rem;
		color: var(--muted);
		word-break: break-all;
		max-width: 120px;
	}

	.total-row td {
		font-weight: 700;
		font-size: 1rem;
		border-top: 2px solid var(--border);
		border-bottom: none;
		color: var(--accent);
	}

	.summary-footer {
		margin-top: auto;
	}

	.back-btn {
		width: 100%;
		min-height: 52px;
		border: 1px solid var(--border);
		border-radius: 10px;
		background: var(--surface);
		color: var(--text);
		font-size: 1rem;
		cursor: pointer;
	}
</style>
