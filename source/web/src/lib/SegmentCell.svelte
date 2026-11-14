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
  const tone = {
    not_started: "bg-neutral-bg",
    in_progress: "bg-brand-tint shadow-[inset_0_0_0_2px_var(--color-accent)]",
    awaiting_acceptance: "bg-warning-bg text-warning-fg",
    accepted: "bg-success-bg text-success-fg",
  } as const;
  const text = $derived(
    segment?.state === "accepted" && segment.score !== undefined ? String(segment.score) : "",
  );
</script>

{#if segment}
  <span
    class="inline-block min-h-6 min-w-10 rounded-sm px-2 text-center align-middle leading-6 font-bold {tone[
      segment.state
    ]}"
    title={label}
    aria-label={label}
    role="img">{text}</span
  >
{/if}
