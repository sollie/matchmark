<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "./utils";

  let {
    title = "Something went wrong",
    message,
    class: className,
    action,
  }: { title?: string; message?: string; class?: string; action?: Snippet } = $props();
</script>

<div
  role="alert"
  class={cn(
    "flex flex-col items-start gap-2 rounded-lg border border-danger bg-danger-bg p-4 text-danger-fg",
    className,
  )}
>
  <p class="m-0 font-semibold">{title}</p>
  {#if message}<p class="m-0">{message}</p>{/if}
  {#if action}<div class="mt-1">{@render action()}</div>{/if}
</div>
