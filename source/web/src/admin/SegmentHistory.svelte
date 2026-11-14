<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import { segmentLabel } from "../lib/rulebook.svelte";
  import Badge, { type BadgeTone } from "../lib/ui/Badge.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import LoadingState from "../lib/ui/LoadingState.svelte";
  import NativeSelect from "../lib/ui/NativeSelect.svelte";
  import Table from "../lib/ui/Table.svelte";

  type History = components["schemas"]["SegmentHistory"];

  let { matchId, version, course }: { matchId: number; version: number; course: string } = $props();

  let history = $state<History | null>(null);
  let error = $state("");
  let scorer = $state("");
  let shooter = $state("");
  let segment = $state("");
  let scorerOptions = $state<string[]>([]);
  let shooterOptions = $state<string[]>([]);
  let segmentOptions = $state<number[]>([]);

  const labels: Record<string, string> = {
    locked: "Locked",
    accepted: "Accepted",
    disputed: "Disputed",
    accepted_all: "Accepted (all)",
    unlocked: "Unlocked",
  };

  const tones: Record<string, BadgeTone> = {
    locked: "info",
    accepted: "success",
    disputed: "danger",
    accepted_all: "success",
    unlocked: "warning",
  };

  const union = <T,>(known: T[], more: T[]) => [...new Set([...known, ...more])].sort();

  async function load() {
    const result = await api.GET("/api/matches/{matchID}/segment-events", {
      params: {
        path: { matchID: matchId },
        query: {
          scorer: scorer || undefined,
          shooter: shooter || undefined,
          segment: segment ? Number(segment) : undefined,
        },
      },
    });
    if (!result.data) {
      error = failure(result);
      return;
    }
    error = "";
    history = result.data;
    scorerOptions = union(
      scorerOptions,
      result.data.events.map((e) => e.scorer_name).filter(Boolean),
    );
    shooterOptions = union(
      shooterOptions,
      result.data.events.map((e) => e.shooter_name),
    );
    segmentOptions = union(
      segmentOptions,
      result.data.events.map((e) => e.segment),
    );
  }

  $effect(() => {
    void version;
    void scorer;
    void shooter;
    void segment;
    void load();
  });

  const actor = (e: components["schemas"]["SegmentHistoryEvent"]) =>
    e.actor_type === "device" ? `Device (${e.scorer_name || "unknown"})` : e.actor_name;
</script>

<section class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card">
  <h2 class="m-0 text-lg">Dispute history</h2>
  <div class="grid gap-3 sm:grid-cols-3">
    <Field label="Scorer">
      <NativeSelect aria-label="Filter by scorer" bind:value={scorer}>
        <option value="">All scorers</option>
        {#each scorerOptions as name (name)}<option value={name}>{name}</option>{/each}
      </NativeSelect>
    </Field>
    <Field label="Shooter">
      <NativeSelect aria-label="Filter by shooter" bind:value={shooter}>
        <option value="">All shooters</option>
        {#each shooterOptions as name (name)}<option value={name}>{name}</option>{/each}
      </NativeSelect>
    </Field>
    <Field label="Segment">
      <NativeSelect aria-label="Filter by segment" bind:value={segment}>
        <option value="">All segments</option>
        {#each segmentOptions as n (n)}<option value={String(n)}>{segmentLabel(course, n)}</option
          >{/each}
      </NativeSelect>
    </Field>
  </div>
  {#if error}<ErrorState title={error} />{/if}
  {#if history}
    <h3 class="m-0 text-base">Disputes per scorer</h3>
    <Table label="Disputes per scorer" class="scorers">
      <thead><tr><th>Scorer</th><th>Locks</th><th>Disputes</th></tr></thead>
      <tbody>
        {#each history.scorers as s (s.scorer_name)}
          <tr><td>{s.scorer_name}</td><td>{s.locks}</td><td>{s.disputes}</td></tr>
        {:else}
          <tr><td colspan="3" class="text-muted">No scorer events.</td></tr>
        {/each}
      </tbody>
    </Table>
    <h3 class="m-0 text-base">Segment events</h3>
    <Table label="Segment events" class="events">
      <thead>
        <tr
          ><th>Time</th><th>Segment</th><th>Event</th><th>Shooter</th><th>Scorer</th><th>By</th></tr
        >
      </thead>
      <tbody>
        {#each history.events as e (e.id)}
          <tr>
            <td class="whitespace-nowrap">{new Date(e.time).toLocaleString()}</td>
            <td>{segmentLabel(course, e.segment)}</td>
            <td><Badge tone={tones[e.event]}>{labels[e.event]}</Badge></td>
            <td>{e.shooter_name}</td>
            <td>{e.scorer_name || "Admin"}</td>
            <td>{actor(e)}</td>
          </tr>
        {:else}
          <tr><td colspan="6" class="text-muted">No events match.</td></tr>
        {/each}
      </tbody>
    </Table>
  {:else if !error}
    <LoadingState />
  {/if}
</section>
