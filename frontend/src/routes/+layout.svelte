<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import { auth } from '$lib/stores/auth';
	import { restoreSession } from '$lib/api/client';
	import { installIOSFallback } from '$lib/sync/queue';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();

	function isPublic(path: string): boolean {
		return path === '/login' || path === '/live' || path.startsWith('/live/');
	}

	onMount(() => {
		let unsubscribeAuth: (() => void) | null = null;
		let removeIOSFallback: (() => void) | null = null;

		async function init() {
			if (!isPublic(get(page).url.pathname)) {
				await restoreSession();
				const state = get(auth);
				if (!state.accessToken) {
					goto('/login');
					return;
				}
				if (get(page).url.pathname === '/') {
					if (state.user?.role === 'admin') {
						goto('/admin');
					} else {
						goto('/score/select');
					}
				}
			}

			unsubscribeAuth = auth.subscribe((state) => {
				if (!isPublic(get(page).url.pathname) && !state.accessToken) {
					goto('/login');
				}
			});

			removeIOSFallback = installIOSFallback();
		}

		init();

		return () => {
			unsubscribeAuth?.();
			removeIOSFallback?.();
		};
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}
