<script lang="ts">
  import MenuShell from "./MenuShell.svelte";
  import ThemeSelector from "./ThemeSelector.svelte";

  let { adminLink }: { adminLink: boolean } = $props();
</script>

<MenuShell label="Menu">
  {#snippet children()}
    <ThemeSelector />
    {#if adminLink}<a href="/admin">Admin</a>{/if}
  {/snippet}
</MenuShell>
