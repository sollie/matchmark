<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiFetch } from '$lib/api/client';

	const WEAPON_CLASSES: Record<string, string[]> = {
		'1500': [
			'Revolver 1500',
			'Pistol 1500',
			'Revolver Optical Sight 1500',
			'Pistol Optical Sight 1500'
		],
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

	interface ShooterRow {
		key: number;
		name: string;
		wa_id: string;
		club: string;
		lane_number: string;
		discipline: string;
		weapon_class: string;
	}

	let compName = $state('');
	let compDate = $state('');
	let lanesTotal = $state('');
	let spectatorEnabled = $state(false);

	let shooterRows = $state<ShooterRow[]>([]);
	let nextKey = $state(0);

	let saving = $state(false);
	let error = $state('');

	function addShooter() {
		shooterRows = [
			...shooterRows,
			{ key: nextKey++, name: '', wa_id: '', club: '', lane_number: '', discipline: '1500', weapon_class: WEAPON_CLASSES['1500'][0] }
		];
	}

	function removeShooter(key: number) {
		shooterRows = shooterRows.filter((r) => r.key !== key);
	}

	function onDisciplineChange(key: number) {
		shooterRows = shooterRows.map((r) => {
			if (r.key !== key) return r;
			return { ...r, weapon_class: WEAPON_CLASSES[r.discipline][0] };
		});
	}

	async function submit() {
		error = '';
		saving = true;
		try {
			const compRes = await apiFetch('/api/competitions', {
				method: 'POST',
				body: JSON.stringify({
					name: compName,
					date: compDate,
					lanes_total: parseInt(lanesTotal, 10),
					spectator_enabled: spectatorEnabled
				})
			});
			if (!compRes.ok) {
				const d = await compRes.json().catch(() => ({}));
				error = d.error ?? 'Failed to create competition';
				return;
			}
			const comp = await compRes.json();

			for (const row of shooterRows) {
				const sRes = await apiFetch(`/api/competitions/${comp.id}/shooters`, {
					method: 'POST',
					body: JSON.stringify({
						name: row.name,
						wa_id: row.wa_id,
						club: row.club || undefined,
						lane_number: parseInt(row.lane_number, 10),
						discipline: row.discipline,
						weapon_class: row.weapon_class
					})
				});
				if (!sRes.ok) {
					const d = await sRes.json().catch(() => ({}));
					error = `Lane ${row.lane_number} (${row.name}): ${d.error ?? 'Failed to add shooter'}`;
					return;
				}
			}

			goto(`/admin/competitions/${comp.id}`);
		} catch {
			error = 'Network error';
		} finally {
			saving = false;
		}
	}

	let canSubmit = $derived(
		compName.trim() !== '' &&
		compDate !== '' &&
		parseInt(lanesTotal, 10) > 0
	);
</script>

<main class="admin-page">
	<header class="page-header">
		<div class="breadcrumb">
			<a href="/admin">Admin</a>
			<span>/</span>
			<a href="/admin/competitions">Competitions</a>
			<span>/ New</span>
		</div>
	</header>

	<form onsubmit={(e) => { e.preventDefault(); submit(); }}>
		<section class="form-section">
			<h2>Competition details</h2>

			<label>
				Name
				<input type="text" bind:value={compName} required disabled={saving} />
			</label>
			<label>
				Date
				<input type="date" bind:value={compDate} required disabled={saving} />
			</label>
			<label>
				Total lanes
				<input type="number" min="1" bind:value={lanesTotal} required disabled={saving} />
			</label>
			<label class="checkbox-label">
				<input type="checkbox" bind:checked={spectatorEnabled} disabled={saving} />
				Enable spectator live view
			</label>
		</section>

		<section class="form-section">
			<div class="section-header">
				<h2>Shooters</h2>
				<button type="button" class="btn-sm" onclick={addShooter} disabled={saving}>+ Add shooter</button>
			</div>

			{#if shooterRows.length === 0}
				<p class="muted">No shooters added yet. You can add them now or after creating the competition.</p>
			{/if}

			{#each shooterRows as row (row.key)}
				<div class="shooter-row">
					<div class="shooter-fields">
						<label>
							Name
							<input type="text" bind:value={row.name} required disabled={saving} />
						</label>
						<label>
							WA-ID
							<input type="text" bind:value={row.wa_id} required disabled={saving} />
						</label>
						<label>
							Club
							<input type="text" bind:value={row.club} disabled={saving} />
						</label>
						<label>
							Lane
							<input type="number" min="1" bind:value={row.lane_number} required disabled={saving} />
						</label>
						<label>
							Discipline
							<select bind:value={row.discipline} onchange={() => onDisciplineChange(row.key)} disabled={saving}>
								<option value="1500">1500</option>
								<option value="60">60-shot</option>
								<option value="48">48-shot</option>
							</select>
						</label>
						<label>
							Weapon class
							<select bind:value={row.weapon_class} disabled={saving}>
								{#each WEAPON_CLASSES[row.discipline] as wc (wc)}
									<option value={wc}>{wc}</option>
								{/each}
							</select>
						</label>
					</div>
					<button type="button" class="btn-remove" onclick={() => removeShooter(row.key)} disabled={saving} aria-label="Remove shooter">✕</button>
				</div>
			{/each}
		</section>

		{#if error}
			<p class="error">{error}</p>
		{/if}

		<div class="form-actions">
			<button type="button" class="btn-secondary" onclick={() => goto('/admin/competitions')} disabled={saving}>Cancel</button>
			<button type="submit" class="btn-primary" disabled={saving || !canSubmit}>
				{saving ? 'Creating…' : 'Create competition'}
			</button>
		</div>
	</form>
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

	form {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.form-section {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.875rem;
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	h2 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
		color: var(--text);
	}

	label {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		font-size: 0.875rem;
		color: var(--text);
	}

	.checkbox-label {
		flex-direction: row;
		align-items: center;
		gap: 0.5rem;
	}

	.checkbox-label input[type='checkbox'] {
		width: 18px;
		height: 18px;
		accent-color: var(--accent);
	}

	input[type='text'],
	input[type='number'],
	input[type='date'],
	select {
		padding: 0.625rem 0.875rem;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--bg);
		color: var(--text);
		font-size: 1rem;
	}

	input:focus,
	select:focus {
		outline: 2px solid var(--accent);
		outline-offset: 2px;
	}

	.shooter-row {
		display: flex;
		gap: 0.75rem;
		align-items: flex-start;
		padding: 1rem;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: 8px;
	}

	.shooter-fields {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
		gap: 0.75rem;
		flex: 1;
	}

	.btn-remove {
		flex-shrink: 0;
		margin-top: 1.5rem;
		background: none;
		border: 1px solid rgba(220, 38, 38, 0.4);
		border-radius: 6px;
		color: #f87171;
		font-size: 0.875rem;
		width: 32px;
		height: 32px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.muted {
		color: var(--muted);
		font-size: 0.875rem;
		margin: 0;
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

	.form-actions {
		display: flex;
		gap: 0.75rem;
		justify-content: flex-end;
	}

	.btn-primary {
		padding: 0.625rem 1.5rem;
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
	}

	.btn-sm:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
