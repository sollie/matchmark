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

  const label = $derived.by(() => {
    if (!stage) return "";
    const base = prefix ? `Stage ${stage.stage}: ${names[stage.state]}` : names[stage.state];
    return stage.state === "accepted" && stage.score !== undefined
      ? `${base}, ${stage.score}`
      : base;
  });
  const text = $derived(
    stage?.state === "accepted" && stage.score !== undefined ? String(stage.score) : "",
  );
</script>

{#if stage}
  <span class="cell {stage.state}" title={label} aria-label={label} role="img">{text}</span>
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
