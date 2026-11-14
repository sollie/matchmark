<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "./utils";

  let {
    title,
    heading = "h2",
    description,
    class: className,
    action,
  }: {
    title: string;
    heading?: "h1" | "h2" | "h3";
    description?: string;
    class?: string;
    action?: Snippet;
  } = $props();
</script>

<div
  class={cn(
    "flex flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-center",
    className,
  )}
>
  <svelte:element this={heading} class="m-0 text-lg font-semibold">{title}</svelte:element>
  {#if description}<p class="m-0 max-w-prose text-muted">{description}</p>{/if}
  {#if action}<div class="mt-2">{@render action()}</div>{/if}
</div>
