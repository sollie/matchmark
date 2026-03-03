<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	type Competition = {
		id: number;
		name: string;
		date: string;
		status: string;
	};

	let competitions: Competition[] = $state([]);
	let loading = $state(true);
	let error = $state('');

	onMount(() => {
		async function load() {
			try {
				const res = await fetch('/api/spectator/competitions');
				if (!res.ok) {
					error = 'Failed to load competitions.';
					return;
				}
				competitions = await res.json();
			} catch {
				error = 'Network error. Please try again.';
			} finally {
				loading = false;
			}
		}
		load();
	});

	function select(id: number) {
		goto(`/live/${id}`);
	}
</script>

<div class="page">
	<h1>Live Scoreboard</h1>

	{#if loading}
		<p class="muted">Loading competitions...</p>
	{:else if error}
		<p class="error">{error}</p>
	{:else if competitions.length === 0}
		<p class="muted">No competitions are currently available for public viewing.</p>
	{:else}
		<div class="cards">
			{#each competitions as comp (comp.id)}
				<button class="card" onclick={() => select(comp.id)}>
					<span class="card-name">{comp.name}</span>
					<span class="card-date">{comp.date}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.page {
		max-width: 600px;
		margin: 2rem auto;
		padding: 0 1rem;
	}

	h1 {
		color: var(--text);
		margin-bottom: 1.5rem;
	}

	.muted {
		color: var(--muted);
	}

	.error {
		color: #e74c3c;
	}

	.cards {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.card {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.25rem;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 8px;
		padding: 1.25rem 1.5rem;
		cursor: pointer;
		text-align: left;
		width: 100%;
		transition: border-color 0.15s;
	}

	.card:hover,
	.card:focus {
		border-color: var(--accent);
		outline: none;
	}

	.card-name {
		font-size: 1.1rem;
		font-weight: 600;
		color: var(--text);
	}

	.card-date {
		font-size: 0.9rem;
		color: var(--muted);
	}
</style>
