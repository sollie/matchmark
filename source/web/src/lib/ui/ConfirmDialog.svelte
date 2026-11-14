<script lang="ts">
  import Button from "./Button.svelte";
  import Dialog from "./Dialog.svelte";

  let {
    title,
    description,
    confirmLabel,
    cancelLabel = "Cancel",
    tone = "danger",
    onconfirm,
    oncancel,
  }: {
    title: string;
    description: string;
    confirmLabel: string;
    cancelLabel?: string;
    tone?: "danger" | "primary";
    onconfirm: () => void | Promise<void>;
    oncancel: () => void;
  } = $props();
</script>

<Dialog {title} {description} onclose={oncancel}>
  <div class="flex flex-wrap justify-end gap-2">
    <Button variant="secondary" onclick={oncancel}>{cancelLabel}</Button>
    <Button variant={tone} onclick={onconfirm}>{confirmLabel}</Button>
  </div>
</Dialog>
