<script lang="ts">
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import { apiFetch } from '$lib/api/client';
	import { auth } from '$lib/stores/auth';

	const compId = parseInt(get(page).params.id ?? '0', 10);

	let downloading = $state<string | null>(null);
	let error = $state('');

	async function download(format: 'pdf' | 'csv' | 'json') {
		error = '';
		downloading = format;
		try {
			const token = get(auth).accessToken;
			const res = await apiFetch(`/api/competitions/${compId}/export/${format}`);
			if (!res.ok) {
				error = `Export failed (${res.status})`;
				return;
			}
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			const ext = format === 'json' ? 'json' : format === 'csv' ? 'csv' : 'pdf';
			a.download = `competition-${compId}.${ext}`;
			a.click();
			URL.revokeObjectURL(url);
			void token;
		} catch {
			error = 'Download failed — check your connection';
		} finally {
			downloading = null;
		}
	}
</script>

<main class="admin-page">
	<header class="page-header">
		<div class="breadcrumb">
			<a href="/admin">Admin</a>
			<span>/</span>
			<a href="/admin/competitions">Competitions</a>
			<span>/</span>
			<a href="/admin/competitions/{compId}">#{compId}</a>
			<span>/ Export</span>
		</div>
	</header>

	<section class="export-section">
		<h2>Export results</h2>
		<p class="muted">Download the final results in your preferred format.</p>

		{#if error}
			<p class="error">{error}</p>
		{/if}

		<div class="export-buttons">
			<button
				class="export-btn"
				onclick={() => download('pdf')}
				disabled={downloading !== null}
			>
				<span class="export-icon">PDF</span>
				<span class="export-label">Score cards</span>
				<span class="export-sub">One page per shooter, WA1500 layout</span>
				{#if downloading === 'pdf'}<span class="export-loading">Downloading…</span>{/if}
			</button>

			<button
				class="export-btn"
				onclick={() => download('csv')}
				disabled={downloading !== null}
			>
				<span class="export-icon">CSV</span>
				<span class="export-label">Spreadsheet</span>
				<span class="export-sub">Import into Excel or Google Sheets</span>
				{#if downloading === 'csv'}<span class="export-loading">Downloading…</span>{/if}
			</button>

			<button
				class="export-btn"
				onclick={() => download('json')}
				disabled={downloading !== null}
			>
				<span class="export-icon">JSON</span>
				<span class="export-label">Raw data</span>
				<span class="export-sub">Full results with per-ring breakdown</span>
				{#if downloading === 'json'}<span class="export-loading">Downloading…</span>{/if}
			</button>
		</div>
	</section>
</main>

<style>
	.admin-page {
		min-height: 100dvh;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.page-header {
		display: flex;
		align-items: center;
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

	.export-section {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	h2 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
	}

	.muted {
		margin: 0;
		color: var(--muted);
		font-size: 0.875rem;
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

	.export-buttons {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.export-btn {
		display: grid;
		grid-template-columns: 56px 1fr;
		grid-template-rows: auto auto;
		column-gap: 1rem;
		align-items: center;
		padding: 1rem;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: 10px;
		cursor: pointer;
		text-align: left;
		min-height: 72px;
	}

	.export-btn:active:not(:disabled) {
		background: var(--surface-active);
	}

	.export-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.export-icon {
		grid-row: 1 / 3;
		font-size: 0.75rem;
		font-weight: 800;
		letter-spacing: 0.05em;
		color: var(--accent);
		background: rgba(240, 192, 64, 0.12);
		border: 1px solid rgba(240, 192, 64, 0.3);
		border-radius: 6px;
		padding: 0.375rem 0.25rem;
		text-align: center;
	}

	.export-label {
		font-size: 0.9375rem;
		font-weight: 600;
		color: var(--text);
	}

	.export-sub {
		font-size: 0.8125rem;
		color: var(--muted);
	}

	.export-loading {
		grid-column: 2;
		font-size: 0.8125rem;
		color: var(--accent);
	}
</style>
