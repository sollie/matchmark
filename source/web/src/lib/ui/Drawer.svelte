<script lang="ts">
  import { Dialog } from "bits-ui";
  import X from "@lucide/svelte/icons/x";
  import type { Snippet } from "svelte";
  import { cn } from "./utils";

  let {
    open = $bindable(false),
    title,
    side = "bottom",
    class: className,
    children,
  }: {
    open?: boolean;
    title: string;
    side?: "bottom" | "left";
    class?: string;
    children: Snippet;
  } = $props();
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay
      class="fixed inset-0 z-50 bg-overlay motion-safe:data-[state=open]:animate-[fade-in_150ms_ease-out]"
    />
    <Dialog.Content
      class={cn(
        "fixed z-51 overflow-auto border-border bg-surface p-4 text-fg shadow-overlay",
        side === "bottom" &&
          "inset-x-0 bottom-0 max-h-[85dvh] rounded-t-xl border-t pb-[max(1rem,env(safe-area-inset-bottom))] motion-safe:data-[state=open]:animate-[drawer-up_200ms_ease-out]",
        side === "left" &&
          "inset-y-0 left-0 w-[min(20rem,85vw)] border-r motion-safe:data-[state=open]:animate-[drawer-left_200ms_ease-out]",
        className,
      )}
    >
      <Dialog.Title class="m-0 mb-3 pr-10 text-lg">{title}</Dialog.Title>
      <Dialog.Description class="sr-only">{title}</Dialog.Description>
      {@render children()}
      <Dialog.Close
        aria-label="Dismiss"
        class="absolute top-2 right-2 inline-flex size-11 cursor-pointer items-center justify-center rounded-md text-muted outline-offset-2 hover:bg-surface-alt hover:text-fg focus-visible:outline-3 focus-visible:outline-focus"
      >
        <X class="size-5" aria-hidden="true" />
      </Dialog.Close>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
