<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "./utils";

  let {
    label,
    tableLabel,
    class: className,
    children,
  }: { label: string; tableLabel?: string; class?: string; children: Snippet } = $props();
</script>

<div
  class="relative overflow-x-auto rounded-lg border border-border bg-surface"
  role="region"
  aria-label={label}
>
  <table
    aria-label={tableLabel}
    class={cn(
      "w-full border-collapse text-left [&_td]:border-t [&_td]:border-border [&_td]:px-3 [&_td]:py-2 [&_th]:bg-surface-alt [&_th]:px-3 [&_th]:py-2 [&_th]:text-sm [&_th]:font-semibold [&_th]:text-muted [&_tr:hover_td]:bg-surface-alt",
      className,
    )}
  >
    {@render children()}
  </table>
</div>
