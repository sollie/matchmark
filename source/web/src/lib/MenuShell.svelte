<script lang="ts">
  import type { Snippet } from "svelte";
  import { router } from "./router.svelte";
  import AccountIcon from "./AccountIcon.svelte";

  let {
    label,
    current = false,
    children,
  }: { label: string; current?: boolean; children: Snippet<[() => void]> } = $props();

  let open = $state(false);
  let root: HTMLElement | undefined = $state();
  let trigger: HTMLButtonElement | undefined = $state();

  $effect(() => {
    void router.path;
    open = false;
  });

  function close() {
    open = false;
  }

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
    aria-label={label}
    title={label}
    aria-haspopup="menu"
    aria-expanded={open}
    aria-current={current ? "page" : undefined}
    onclick={() => (open = !open)}
  >
    <AccountIcon />
  </button>
  {#if open}
    <div class="account-panel">
      {@render children(close)}
    </div>
  {/if}
</div>
