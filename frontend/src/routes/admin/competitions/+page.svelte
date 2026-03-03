<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { apiFetch } from '$lib/api/client';

	interface Competition {
		id: number;
		name: string;
		date: string;
		lanes_total: number;
		status: 'draft' | 'active' | 'finished';
		spectator_enabled: boolean;
	}

	let competitions = $state<Competition[]>([]);
	let loadError = $state('');

	async function load() {
		const res = await apiFetch('/api/competitions');
		if (!res.ok) { loadError = 'Failed to load competitions'; return; }
		competitions = await res.json();
	}

	async function setStatus(c: Competition, status: Competition['status']) {
		const res = await apiFetch(`/api/competitions/${c.id}`, {
			method: 'PUT',
			body: JSON.stringify({ status })
		});
		if (res.ok) await load();
	}

	onMount(() => { load(); });
</script>

<main class="admin-page">
	<header class="page-header">
		<div class="breadcrumb">
			<a href="/admin">Admin</a>
			<span>/ Competitions</span>
		</div>
		<button class="btn-primary" onclick={() => goto('/admin/competitions/new')}>+ New</button>
	</header>

	{#if loadError}
		<p class="error">{loadError}</p>
	{/if}

	{#if competitions.length === 0 && !loadError}
		<p class="muted">No competitions yet.</p>
	{/if}

	<div class="comp-list">
		{#each competitions as c (c.id)}
			<div class="comp-card">
				<a href="/admin/competitions/{c.id}" class="comp-info">
					<span class="comp-name">{c.name}</span>
					<span class="comp-meta">{c.date} · {c.lanes_total} lanes</span>
				</a>
				<div class="comp-actions">
					<span class="badge status-{c.status}">{c.status}</span>
					{#if c.status === 'draft'}
						<button class="btn-sm" onclick={() => setStatus(c, 'active')}>Activate</button>
					{:else if c.status === 'active'}
						<button class="btn-sm" onclick={() => setStatus(c, 'finished')}>Finish</button>
					{/if}
					<a href="/admin/competitions/{c.id}/export" class="btn-sm">Export</a>
				</div>
			</div>
		{/each}
	</div>
</main>

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
	}

	.breadcrumb {
		font-size: 0.9375rem;
	}

	.breadcrumb a {
		color: var(--accent);
		text-decoration: none;
	}

	.breadcrumb span {
		color: var(--muted);
		margin-left: 0.25rem;
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

	.muted {
		color: var(--muted);
		font-size: 0.875rem;
	}

	.comp-list {
		display: flex;
		flex-direction: column;
		gap: 0.625rem;
	}

	.comp-card {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 1rem;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
	}

	.comp-info {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		text-decoration: none;
		flex: 1;
		min-width: 0;
	}

	.comp-name {
		font-size: 1rem;
		font-weight: 600;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.comp-meta {
		font-size: 0.8125rem;
		color: var(--muted);
	}

	.comp-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-shrink: 0;
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
</style>
