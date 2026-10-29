<script lang="ts">
  import type { components } from "./api/schema";

  type SegmentScore = components["schemas"]["SegmentScore"];

  let { segment, prefix = true }: { segment: SegmentScore | undefined; prefix?: boolean } =
    $props();

  const names = {
    not_started: "not started",
    in_progress: "in progress",
    awaiting_acceptance: "awaiting acceptance",
    accepted: "accepted",
  } as const;

  const label = $derived.by(() => {
    if (!segment) return "";
    const base = prefix
      ? `Segment ${segment.segment}: ${names[segment.state]}`
      : names[segment.state];
    return segment.state === "accepted" && segment.score !== undefined
      ? `${base}, ${segment.score}`
      : base;
  });
  const text = $derived(
    segment?.state === "accepted" && segment.score !== undefined ? String(segment.score) : "",
  );
</script>

{#if segment}
  <span class="cell {segment.state}" title={label} aria-label={label} role="img">{text}</span>
{/if}

<style>
  .cell {
    display: inline-block;
    min-width: 2.5em;
    min-height: 1.5em;
    padding: 0 var(--space-2);
    border-radius: var(--radius-sm);
    text-align: center;
    font-weight: 700;
    line-height: 1.5;
    vertical-align: middle;
  }
  .not_started {
    background: var(--neutral-bg);
  }
  .in_progress {
    background: var(--brand-tint);
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .awaiting_acceptance {
    background: var(--warning-bg);
    color: var(--warning-fg);
  }
  .accepted {
    background: var(--success-bg);
    color: var(--success-fg);
  }
</style>
