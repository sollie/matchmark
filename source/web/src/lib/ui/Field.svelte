<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "./utils";

  let {
    label,
    hint,
    error,
    class: className,
    children,
  }: {
    label: string;
    hint?: string;
    error?: string;
    class?: string;
    children: Snippet;
  } = $props();
</script>

<div class={cn("flex flex-col gap-1", className)}>
  <label class="flex flex-col gap-1">
    <span class="text-sm font-medium text-fg">{label}</span>
    {@render children()}
    {#if hint}<span class="text-sm text-muted">{hint}</span>{/if}
  </label>
  {#if error}<span class="text-sm text-danger-fg" role="alert">{error}</span>{/if}
</div>
