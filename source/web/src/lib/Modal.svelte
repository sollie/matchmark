<script lang="ts">
  import { Dialog } from "bits-ui";
  import type { Snippet } from "svelte";

  let { title, onclose, children }: { title: string; onclose: () => void; children: Snippet } =
    $props();
</script>

<Dialog.Root
  open={true}
  onOpenChange={(open) => {
    if (!open) onclose();
  }}
>
  <Dialog.Portal>
    <Dialog.Overlay class="modal-overlay" />
    <Dialog.Content class="modal-content">
      <Dialog.Title class="modal-title">{title}</Dialog.Title>
      <Dialog.Description class="sr-only">{title}</Dialog.Description>
      {@render children()}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
