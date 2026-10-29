<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";

  type History = components["schemas"]["SegmentHistory"];

  let { matchId, version }: { matchId: number; version: number } = $props();

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

<section class="card">
  <h2>Dispute history</h2>
  <div class="row">
    <label>
      Scorer
      <select aria-label="Filter by scorer" bind:value={scorer}>
        <option value="">All scorers</option>
        {#each scorerOptions as name (name)}<option value={name}>{name}</option>{/each}
      </select>
    </label>
    <label>
      Shooter
      <select aria-label="Filter by shooter" bind:value={shooter}>
        <option value="">All shooters</option>
        {#each shooterOptions as name (name)}<option value={name}>{name}</option>{/each}
      </select>
    </label>
    <label>
      Segment
      <select aria-label="Filter by segment" bind:value={segment}>
        <option value="">All segments</option>
        {#each segmentOptions as n (n)}<option value={String(n)}>Segment {n}</option>{/each}
      </select>
    </label>
  </div>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if history}
    <h3>Disputes per scorer</h3>
    <div class="scroll">
      <table class="scorers">
        <thead><tr><th>Scorer</th><th>Locks</th><th>Disputes</th></tr></thead>
        <tbody>
          {#each history.scorers as s (s.scorer_name)}
            <tr><td>{s.scorer_name}</td><td>{s.locks}</td><td>{s.disputes}</td></tr>
          {:else}
            <tr><td colspan="3" class="muted">No scorer events.</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
    <h3>Segment events</h3>
    <div class="scroll">
      <table class="events">
        <thead
          ><tr
            ><th>Time</th><th>Segment</th><th>Event</th><th>Shooter</th><th>Scorer</th><th>By</th
            ></tr
          ></thead
        >
        <tbody>
          {#each history.events as e (e.id)}
            <tr>
              <td>{new Date(e.time).toLocaleString()}</td>
              <td>{e.segment}</td>
              <td>{labels[e.event]}</td>
              <td>{e.shooter_name}</td>
              <td>{e.scorer_name || "Admin"}</td>
              <td>{actor(e)}</td>
            </tr>
          {:else}
            <tr><td colspan="6" class="muted">No events match.</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else if !error}
    <p class="muted">Loading…</p>
  {/if}
</section>
