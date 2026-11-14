<script lang="ts">
  import { Toaster } from "svelte-sonner";
</script>

<Toaster
  position="bottom-center"
  closeButton
  toastOptions={{
    unstyled: true,
    classes: {
      toast:
        "flex w-[min(24rem,calc(100vw-1.5rem))] items-start gap-3 rounded-lg border border-border bg-surface p-3 text-base text-fg shadow-overlay",
      title: "font-semibold",
      description: "text-sm text-muted",
      success: "border-l-4 border-l-success-fg",
      error: "border-l-4 border-l-danger",
      warning: "border-l-4 border-l-warning-fg",
      info: "border-l-4 border-l-brand",
      closeButton:
        "absolute top-1 right-1 inline-flex size-8 cursor-pointer items-center justify-center rounded-md text-muted hover:bg-surface-alt",
    },
  }}
/>
