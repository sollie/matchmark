<script lang="ts">
  import { onMount, type Snippet } from "svelte";

  let { title, onclose, children }: { title: string; onclose: () => void; children: Snippet } =
    $props();

  let dialog: HTMLDialogElement;

  onMount(() => {
    dialog.showModal();
  });
</script>

<dialog
  bind:this={dialog}
  aria-labelledby="modal-title"
  {onclose}
  onclick={(e) => {
    if (e.target === dialog) dialog.close();
  }}
>
  <h3 id="modal-title">{title}</h3>
  {@render children()}
</dialog>

<style>
  dialog {
    width: min(32rem, calc(100vw - 2 * var(--space-3)));
    max-height: calc(100vh - 2 * var(--space-3));
    overflow: auto;
    padding: var(--space-4);
    border: 1px solid var(--border, currentColor);
    border-radius: var(--radius, 0.5rem);
    background: var(--surface, Canvas);
    color: var(--text);
  }
  h3 {
    margin-top: 0;
  }
</style>
