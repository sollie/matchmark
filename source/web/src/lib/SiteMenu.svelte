<script lang="ts">
  import MenuShell from "./MenuShell.svelte";
  import ThemeSelector from "./ThemeSelector.svelte";
  import { screenControl } from "./screenControl.svelte";

  let { adminLink, remoteControl = false }: { adminLink: boolean; remoteControl?: boolean } =
    $props();
</script>

<MenuShell label="Menu">
  {#snippet children(close)}
    <ThemeSelector />
    {#if remoteControl}
      {#if screenControl.managed}
        <button
          type="button"
          disabled={screenControl.busy}
          onclick={() => {
            close();
            void screenControl.leave();
          }}>Leave remote control</button
        >
      {:else}
        <button
          type="button"
          disabled={screenControl.busy}
          onclick={() => {
            close();
            void screenControl.enable();
          }}>Enable remote control</button
        >
      {/if}
    {/if}
    {#if adminLink}<a href="/admin">Admin</a>{/if}
  {/snippet}
</MenuShell>
