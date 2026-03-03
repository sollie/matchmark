<script lang="ts">
	import { goto } from '$app/navigation';
	import { setTokens } from '$lib/api/client';

	let username = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function login() {
		error = '';
		loading = true;
		try {
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});
			if (!res.ok) {
				const data = await res.json().catch(() => ({}));
				error = data.error ?? 'Login failed';
				return;
			}
			const data = await res.json();
			setTokens(
				data.access_token,
				data.refresh_token,
				Date.now() + data.expires_in * 1000,
				data.user
			);
			if (data.user.role === 'admin') {
				goto('/admin');
			} else {
				goto('/score/select');
			}
		} catch {
			error = 'Network error — check your connection';
		} finally {
			loading = false;
		}
	}
</script>

<main class="login-page">
	<div class="login-card">
		<h1>MatchMark</h1>
		<p class="subtitle">WA1500 / PPC Scoring</p>
		<form onsubmit={(e) => { e.preventDefault(); login(); }}>
			<label>
				Username
				<input
					type="text"
					autocomplete="username"
					bind:value={username}
					disabled={loading}
					required
				/>
			</label>
			<label>
				Password
				<input
					type="password"
					autocomplete="current-password"
					bind:value={password}
					disabled={loading}
					required
				/>
			</label>
			{#if error}
				<p class="error">{error}</p>
			{/if}
			<button type="submit" disabled={loading || !username || !password}>
				{loading ? 'Signing in…' : 'Sign in'}
			</button>
		</form>
	</div>
</main>

<style>
	.login-page {
		min-height: 100dvh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
	}

	.login-card {
		width: 100%;
		max-width: 380px;
		background: var(--surface);
		border-radius: 12px;
		padding: 2rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	h1 {
		margin: 0;
		font-size: 2rem;
		color: var(--accent);
		text-align: center;
	}

	.subtitle {
		margin: 0;
		text-align: center;
		color: var(--muted);
		font-size: 0.875rem;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	label {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		font-size: 0.875rem;
		color: var(--text);
	}

	input {
		padding: 0.75rem 1rem;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--bg);
		color: var(--text);
		font-size: 1rem;
	}

	input:focus {
		outline: 2px solid var(--accent);
		outline-offset: 2px;
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

	button {
		padding: 0.875rem;
		border: none;
		border-radius: 8px;
		background: var(--accent);
		color: #fff;
		font-size: 1rem;
		font-weight: 600;
		cursor: pointer;
		min-height: 48px;
	}

	button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
