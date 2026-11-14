<script lang="ts">
  import type { HTMLSelectAttributes } from "svelte/elements";
  import { cn } from "./utils";

  let { value = $bindable(), class: className, children, ...rest }: HTMLSelectAttributes = $props();
</script>

<select
  bind:value
  class={cn(
    "min-h-11 w-full min-w-0 cursor-pointer rounded-md border border-border bg-surface px-3 text-base text-fg outline-offset-2 hover:border-muted focus-visible:outline-3 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-50",
    className,
  )}
  {...rest}
>
  {@render children?.()}
</select>
