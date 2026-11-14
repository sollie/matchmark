<script lang="ts">
  import { Dialog } from "bits-ui";
  import X from "@lucide/svelte/icons/x";
  import type { Snippet } from "svelte";
  import { cn } from "./utils";

  let {
    open = $bindable(true),
    title,
    description,
    onclose,
    onrequestclose,
    closeLabel = "Dismiss",
    autofocus,
    class: className,
    children,
  }: {
    open?: boolean;
    title: string;
    description?: string;
    onclose?: () => void;
    onrequestclose?: () => void;
    closeLabel?: string;
    autofocus?: string;
    class?: string;
    children: Snippet;
  } = $props();

  let content = $state<HTMLElement | null>(null);
</script>

<Dialog.Root
  bind:open={
    () => open,
    (next) => {
      if (!next && onrequestclose) {
        onrequestclose();
        return;
      }
      open = next;
      if (!next) onclose?.();
    }
  }
>
  <Dialog.Portal>
    <Dialog.Overlay
      class="fixed inset-0 z-50 bg-overlay motion-safe:data-[state=open]:animate-[fade-in_150ms_ease-out]"
    />
    <Dialog.Content
      onOpenAutoFocus={(event) => {
        const target = autofocus && content?.querySelector<HTMLElement>(autofocus);
        if (!target) return;
        event.preventDefault();
        target.focus();
      }}
      bind:ref={content}
      class={cn(
        "fixed top-1/2 left-1/2 z-51 max-h-[calc(100dvh-1.5rem)] w-[min(32rem,calc(100vw-1.5rem))] -translate-x-1/2 -translate-y-1/2 overflow-auto rounded-xl border border-border bg-surface p-4 text-fg shadow-overlay motion-safe:data-[state=open]:animate-[dialog-in_150ms_ease-out]",
        className,
      )}
    >
      <Dialog.Title class="m-0 mb-3 pr-10 text-lg">{title}</Dialog.Title>
      <Dialog.Description class={description ? "mb-3 text-muted" : "sr-only"}>
        {description ?? title}
      </Dialog.Description>
      {@render children()}
      <Dialog.Close
        aria-label={closeLabel}
        class="absolute top-2 right-2 inline-flex size-11 cursor-pointer items-center justify-center rounded-md text-muted outline-offset-2 hover:bg-surface-alt hover:text-fg focus-visible:outline-3 focus-visible:outline-focus"
      >
        <X class="size-5" aria-hidden="true" />
      </Dialog.Close>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
