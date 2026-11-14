<script lang="ts">
  import { cn } from "./utils";

  let { label = "Loading", class: className }: { label?: string; class?: string } = $props();
</script>

<div
  role="status"
  class={cn("flex items-center justify-center gap-3 px-4 py-8 text-muted", className)}
>
  <span
    class="size-5 rounded-full border-2 border-border border-t-brand motion-safe:animate-spin"
    aria-hidden="true"
  ></span>
  <span>{label}…</span>
</div>
