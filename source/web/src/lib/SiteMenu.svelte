<script lang="ts">
  import { router } from "./router.svelte";
  import AccountIcon from "./AccountIcon.svelte";
  import ThemeSelector from "./ThemeSelector.svelte";

  let { adminLink }: { adminLink: boolean } = $props();

  let open = $state(false);
  let root: HTMLElement | undefined = $state();
  let trigger: HTMLButtonElement | undefined = $state();

  $effect(() => {
    void router.path;
    open = false;
  });

  function onWindowClick(e: MouseEvent) {
    if (open && root && !root.contains(e.target as Node)) open = false;
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Escape" && open) {
      open = false;
      trigger?.focus();
    }
  }
</script>

<svelte:window onclick={onWindowClick} onkeydown={onKeydown} />

<div class="account-menu" bind:this={root}>
  <button
    bind:this={trigger}
    type="button"
    class="account-link"
    aria-label="Menu"
    title="Menu"
    aria-haspopup="menu"
    aria-expanded={open}
    onclick={() => (open = !open)}
  >
    <AccountIcon />
  </button>
  {#if open}
    <div class="account-panel">
      <ThemeSelector />
      {#if adminLink}<a href="/admin">Admin</a>{/if}
    </div>
  {/if}
</div>
