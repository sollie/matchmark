<script lang="ts">
  import type { components } from "./api/schema";

  type StageScore = components["schemas"]["StageScore"];

  let { stage, prefix = true }: { stage: StageScore | undefined; prefix?: boolean } = $props();

  const names = {
    not_started: "not started",
    in_progress: "in progress",
    awaiting_acceptance: "awaiting acceptance",
    accepted: "accepted",
  } as const;

  const glyphs = {
    not_started: "–",
    in_progress: "…",
    awaiting_acceptance: "⏳",
  } as const;

  const label = $derived.by(() => {
    if (!stage) return "";
    const base = prefix ? `Stage ${stage.stage}: ${names[stage.state]}` : names[stage.state];
    return stage.state === "accepted" && stage.score !== undefined ? `${base}, ${stage.score}` : base;
  });
  const text = $derived.by(() => {
    if (!stage) return "";
    if (stage.state !== "accepted") return glyphs[stage.state];
    return stage.score !== undefined ? String(stage.score) : "✓";
  });
</script>

{#if stage}
  <span class="cell {stage.state}" title={label} aria-label={label} role="img">{text}</span>
{/if}

<style>
  .cell { display: inline-block; min-width: 1.5em; text-align: right; }
  .not_started { color: var(--text-muted); }
  .in_progress { color: var(--accent); font-weight: 600; letter-spacing: 0.05em; }
  .awaiting_acceptance { color: var(--warning-fg); font-style: italic; }
  .accepted { color: var(--text); font-weight: 700; }
</style>
