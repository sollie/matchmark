<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";
  import { cn } from "./utils";

  let { value = $bindable(), class: className, ...rest }: HTMLInputAttributes = $props();
</script>

<input
  bind:value
  class={cn(
    "min-h-11 w-full min-w-0 rounded-md border border-border bg-surface px-3 text-base text-fg outline-offset-2 placeholder:text-muted hover:border-muted focus-visible:outline-3 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-danger",
    className,
  )}
  {...rest}
/>
