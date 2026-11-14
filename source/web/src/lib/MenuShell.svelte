<script lang="ts">
  import { Popover } from "bits-ui";
  import type { Snippet } from "svelte";
  import { router } from "./router.svelte";
  import AccountIcon from "./AccountIcon.svelte";

  let {
    label,
    current = false,
    children,
  }: { label: string; current?: boolean; children: Snippet<[() => void]> } = $props();

  let open = $state(false);

  $effect(() => {
    void router.path;
    open = false;
  });
</script>

<Popover.Root bind:open>
  <Popover.Trigger
    aria-label={label}
    title={label}
    aria-current={current ? "page" : undefined}
    class="ml-2 inline-flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-md border-b-3 border-transparent bg-transparent text-white outline-offset-[-2px] hover:bg-white/10 focus-visible:outline-3 focus-visible:outline-white aria-[current=page]:border-white"
  >
    <AccountIcon />
  </Popover.Trigger>
  <Popover.Portal>
    <Popover.Content
      align="end"
      sideOffset={4}
      class="z-40 flex min-w-56 flex-col rounded-lg border border-border bg-surface p-2 text-fg shadow-overlay motion-safe:data-[state=open]:animate-[fade-in_120ms_ease-out] [&_a]:rounded-md [&_a]:px-3 [&_a]:py-2 [&_a]:text-fg [&_a]:no-underline [&_a:hover]:bg-surface-alt [&_button]:cursor-pointer [&_button]:rounded-md [&_button]:border-0 [&_button]:bg-transparent [&_button]:px-3 [&_button]:py-2 [&_button]:text-left [&_button]:text-fg [&_button:hover:not(:disabled)]:bg-surface-alt [&_h2]:m-0 [&_h2]:mt-2 [&_h2]:px-3 [&_h2]:text-sm [&_h2]:font-semibold [&_h2]:text-muted"
    >
      <div role="group" aria-label={label} class="flex flex-col">
        {@render children(() => (open = false))}
      </div>
    </Popover.Content>
  </Popover.Portal>
</Popover.Root>
