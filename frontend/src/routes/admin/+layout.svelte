<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { get } from 'svelte/store';
	import { auth } from '$lib/stores/auth';

	let { children } = $props();

	onMount(() => {
		const state = get(auth);
		if (state.user?.role !== 'admin') {
			goto('/login');
		}
		return auth.subscribe((s) => {
			if (s.accessToken && s.user?.role !== 'admin') {
				goto('/login');
			}
		});
	});
</script>

{@render children()}
