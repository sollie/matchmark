<script lang="ts">
	import { goto } from '$app/navigation';
	import { signOut } from '$lib/api/client';

	async function handleSignOut() {
		await signOut();
		goto('/login');
	}
</script>

<main class="admin-page">
	<header class="page-header">
		<h1>MatchMark Admin</h1>
		<button class="logout-btn" onclick={handleSignOut}>Sign out</button>
	</header>

	<nav class="dashboard-nav">
		<a href="/admin/users" class="nav-card">
			<span class="nav-title">Users</span>
			<span class="nav-sub">Manage scorer accounts</span>
		</a>
		<a href="/admin/competitions" class="nav-card">
			<span class="nav-title">Competitions</span>
			<span class="nav-sub">Create and monitor competitions</span>
		</a>
	</nav>
</main>

<style>
	.admin-page {
		min-height: 100dvh;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.page-header h1 {
		margin: 0;
		font-size: 1.5rem;
		color: var(--accent);
	}

	.logout-btn {
		background: none;
		border: 1px solid var(--border);
		border-radius: 6px;
		color: var(--muted);
		font-size: 0.875rem;
		padding: 0.375rem 0.75rem;
		cursor: pointer;
	}

	.logout-btn:hover {
		color: var(--text);
		border-color: var(--text);
	}

	.dashboard-nav {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.nav-card {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		padding: 1.25rem;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		text-decoration: none;
		min-height: 72px;
		justify-content: center;
	}

	.nav-card:active {
		background: var(--surface-active);
	}

	.nav-title {
		font-size: 1rem;
		font-weight: 600;
		color: var(--text);
	}

	.nav-sub {
		font-size: 0.8125rem;
		color: var(--muted);
	}
</style>
