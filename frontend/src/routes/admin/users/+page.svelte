<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { apiFetch } from '$lib/api/client';

	interface User {
		id: number;
		username: string;
		role: 'user' | 'admin';
		created_at: string;
	}

	let users = $state<User[]>([]);
	let loadError = $state('');

	let showCreateModal = $state(false);
	let createUsername = $state('');
	let createPassword = $state('');
	let createRole = $state<'user' | 'admin'>('user');
	let createError = $state('');
	let createLoading = $state(false);

	let editUser = $state<User | null>(null);
	let editUsername = $state('');
	let editRole = $state<'user' | 'admin'>('user');
	let editPassword = $state('');
	let editError = $state('');
	let editLoading = $state(false);

	let deleteUser = $state<User | null>(null);
	let deleteError = $state('');
	let deleteLoading = $state(false);

	async function load() {
		const res = await apiFetch('/api/users');
		if (!res.ok) { loadError = 'Failed to load users'; return; }
		users = await res.json();
	}

	onMount(() => { load(); });

	function openCreate() {
		createUsername = '';
		createPassword = '';
		createRole = 'user';
		createError = '';
		showCreateModal = true;
	}

	async function submitCreate() {
		createError = '';
		createLoading = true;
		try {
			const res = await apiFetch('/api/users', {
				method: 'POST',
				body: JSON.stringify({ username: createUsername, password: createPassword, role: createRole })
			});
			if (!res.ok) {
				const d = await res.json().catch(() => ({}));
				createError = d.error ?? 'Failed to create user';
				return;
			}
			showCreateModal = false;
			await load();
		} catch {
			createError = 'Network error';
		} finally {
			createLoading = false;
		}
	}

	function openEdit(u: User) {
		editUser = u;
		editUsername = u.username;
		editRole = u.role;
		editPassword = '';
		editError = '';
	}

	async function submitEdit() {
		if (!editUser) return;
		editError = '';
		editLoading = true;
		try {
			const body: Record<string, string> = { username: editUsername, role: editRole };
			if (editPassword) body.password = editPassword;
			const res = await apiFetch(`/api/users/${editUser.id}`, {
				method: 'PUT',
				body: JSON.stringify(body)
			});
			if (!res.ok) {
				const d = await res.json().catch(() => ({}));
				editError = d.error ?? 'Failed to update user';
				return;
			}
			editUser = null;
			await load();
		} catch {
			editError = 'Network error';
		} finally {
			editLoading = false;
		}
	}

	function openDelete(u: User) {
		deleteUser = u;
		deleteError = '';
	}

	async function submitDelete() {
		if (!deleteUser) return;
		deleteError = '';
		deleteLoading = true;
		try {
			const res = await apiFetch(`/api/users/${deleteUser.id}`, { method: 'DELETE' });
			if (!res.ok) {
				const d = await res.json().catch(() => ({}));
				deleteError = d.error ?? 'Failed to delete user';
				return;
			}
			deleteUser = null;
			await load();
		} catch {
			deleteError = 'Network error';
		} finally {
			deleteLoading = false;
		}
	}
</script>

<main class="admin-page">
	<header class="page-header">
		<div class="breadcrumb">
			<a href="/admin">Admin</a>
			<span>/ Users</span>
		</div>
		<button class="btn-primary" onclick={openCreate}>+ New user</button>
	</header>

	{#if loadError}
		<p class="error">{loadError}</p>
	{/if}

	<div class="table-wrap">
		<table>
			<thead>
				<tr>
					<th>Username</th>
					<th>Role</th>
					<th>Created</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each users as u (u.id)}
					<tr>
						<td>{u.username}</td>
						<td><span class="badge" class:admin={u.role === 'admin'}>{u.role}</span></td>
						<td class="muted">{u.created_at.slice(0, 10)}</td>
						<td class="actions">
							<button class="btn-sm" onclick={() => openEdit(u)}>Edit</button>
							<button class="btn-sm danger" onclick={() => openDelete(u)}>Delete</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</main>

{#if showCreateModal}
	<div class="overlay" role="dialog" aria-modal="true" aria-label="Create user">
		<div class="modal">
			<h2>Create user</h2>
			<form onsubmit={(e) => { e.preventDefault(); submitCreate(); }}>
				<label>
					Username
					<input type="text" bind:value={createUsername} required disabled={createLoading} />
				</label>
				<label>
					Password <span class="muted">(min 16 chars)</span>
					<input type="password" bind:value={createPassword} minlength="16" required disabled={createLoading} />
				</label>
				<label>
					Role
					<select bind:value={createRole} disabled={createLoading}>
						<option value="user">User (scorer)</option>
						<option value="admin">Admin</option>
					</select>
				</label>
				{#if createError}
					<p class="error">{createError}</p>
				{/if}
				<div class="modal-actions">
					<button type="button" class="btn-secondary" onclick={() => showCreateModal = false} disabled={createLoading}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={createLoading || !createUsername || !createPassword}>
						{createLoading ? 'Creating…' : 'Create'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

{#if editUser}
	<div class="overlay" role="dialog" aria-modal="true" aria-label="Edit user">
		<div class="modal">
			<h2>Edit user</h2>
			<form onsubmit={(e) => { e.preventDefault(); submitEdit(); }}>
				<label>
					Username
					<input type="text" bind:value={editUsername} required disabled={editLoading} />
				</label>
				<label>
					Role
					<select bind:value={editRole} disabled={editLoading}>
						<option value="user">User (scorer)</option>
						<option value="admin">Admin</option>
					</select>
				</label>
				<label>
					New password <span class="muted">(leave blank to keep current)</span>
					<input type="password" bind:value={editPassword} minlength="16" disabled={editLoading} />
				</label>
				{#if editError}
					<p class="error">{editError}</p>
				{/if}
				<div class="modal-actions">
					<button type="button" class="btn-secondary" onclick={() => editUser = null} disabled={editLoading}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={editLoading || !editUsername}>
						{editLoading ? 'Saving…' : 'Save'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

{#if deleteUser}
	<div class="overlay" role="dialog" aria-modal="true" aria-label="Delete user">
		<div class="modal">
			<h2>Delete user</h2>
			<p>Delete <strong>{deleteUser.username}</strong>? This cannot be undone.</p>
			{#if deleteError}
				<p class="error">{deleteError}</p>
			{/if}
			<div class="modal-actions">
				<button class="btn-secondary" onclick={() => deleteUser = null} disabled={deleteLoading}>Cancel</button>
				<button class="btn-primary danger" onclick={submitDelete} disabled={deleteLoading}>
					{deleteLoading ? 'Deleting…' : 'Delete'}
				</button>
			</div>
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
	}

	td {
		padding: 0.75rem;
		border-bottom: 1px solid var(--border);
		vertical-align: middle;
	}

	.muted {
		color: var(--muted);
		font-size: 0.875rem;
	}

	.badge {
		display: inline-block;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		background: var(--surface);
		border: 1px solid var(--border);
		color: var(--muted);
		text-transform: uppercase;
	}

	.badge.admin {
		background: rgba(240, 192, 64, 0.15);
		border-color: rgba(240, 192, 64, 0.4);
		color: var(--accent);
	}

	.actions {
		display: flex;
		gap: 0.5rem;
		justify-content: flex-end;
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

	.btn-primary.danger {
		background: #dc2626;
		color: #fff;
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

	.btn-sm.danger {
		border-color: rgba(220, 38, 38, 0.4);
		color: #f87171;
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

	.modal h2 {
		margin: 0;
		font-size: 1.125rem;
	}

	.modal p {
		margin: 0;
		font-size: 0.9375rem;
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
		margin-top: 0.25rem;
	}
</style>
