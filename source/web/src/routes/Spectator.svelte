<script lang="ts">
  import { api } from "../lib/api/client";
  import { clockTime } from "../lib/datetime";
  import StageCell from "../lib/StageCell.svelte";
  import { openStream, type StreamState } from "../lib/sse";
  import type { components } from "../lib/api/schema";

  type Match = components["schemas"]["Match"];
  type Standing = components["schemas"]["Standing"];
  type Snapshot = components["schemas"]["SpectatorSnapshot"];

  let matches = $state<Match[]>([]);
  let selected = $state<number | null>(null);
  let standings = $state<Standing[]>([]);
  let snapshot = $state<Snapshot | null>(null);
  let connection = $state<StreamState>("connecting");
  let updated = $state<Date | null>(null);

  const showNames = $derived(standings.some((s) => s.last_name !== undefined));
  const showClubs = $derived(standings.some((s) => s.club !== undefined));
  const showCountries = $derived(standings.some((s) => s.country !== undefined));
  const showX = $derived(standings.some((s) => s.x_count !== undefined));
  const current = $derived(matches.find((c) => c.id === selected));
  const finished = $derived(current?.status === "finished");
  const acceptance = $derived.by(() => {
    if (!snapshot) return null;
    const stages = snapshot.standings
      .filter((s) => !s.withdrawn)
      .flatMap((s) => s.stages?.filter((st) => st.stage === snapshot?.current_stage) ?? []);
    return { accepted: stages.filter((st) => st.accepted).length, total: stages.length };
  });
  const stageNumbers = $derived(
    [...new Set(standings.flatMap((s) => s.stages?.map((st) => st.stage) ?? []))].sort(
      (a, b) => a - b,
    ),
  );

  async function loadMatches() {
    const { data } = await api.GET("/api/spectator/matches");
    if (!data) return;
    matches = data.matches;
    if (selected === null || !data.matches.some((c) => c.id === selected)) {
      selected = (data.matches.find((c) => c.status === "active") ?? data.matches[0])?.id ?? null;
    }
  }

  $effect(() => {
    void loadMatches();
  });

  $effect(() => {
    if (selected === null) return;
    const stream = openStream({
      url: `/api/spectator/matches/${selected}/stream`,
      onEvent: (event, data) => {
        if (event !== "snapshot") return;
        snapshot = JSON.parse(data);
        standings = snapshot?.standings ?? [];
        updated = new Date();
        const status = snapshot?.status;
        if (status) matches = matches.map((c) => (c.id === selected ? { ...c, status } : c));
      },
      onState: (next) => {
        connection = next;
        if (next === "live") void loadMatches();
      },
    });
    return () => stream.close();
  });

  function rank(i: number): number {
    let r = i;
    while (r > 0 && standings[r - 1].total === standings[i].total) r--;
    return r + 1;
  }
</script>

<main class="spectator">
  <h1>Live standings</h1>
  {#if matches.length === 0}
    <div class="card empty">
      <p>No matches are live yet.</p>
      <p class="muted">Reload this page once a match has started.</p>
    </div>
  {:else}
    <label class="picker">
      Match
      <select bind:value={selected}>
        {#each matches as c (c.id)}
          <option value={c.id}>{c.name} ({c.date}, {c.status})</option>
        {/each}
      </select>
    </label>
    <div class="status card" role="status">
      <div class="progress-info">
        {#if current}<span class="badge neutral">{current.status}</span>{/if}
        {#if finished}
          <strong>Finished</strong>
          <a href={`/results/${selected}`}>View results</a>
        {:else if snapshot && acceptance}
          <strong>Stage {snapshot.current_stage} of {snapshot.total_stages}</strong>
          <span class="muted">{acceptance.accepted}/{acceptance.total} accepted</span>
        {/if}
      </div>
      <div class="connection">
        {#if connection === "stale"}
          <span class="badge danger">Offline</span>
          {#if updated}<span class="muted">Last update {clockTime(updated)}</span>{/if}
        {:else if updated}
          <span class="badge success">Live</span>
          <span class="muted">Updated {clockTime(updated)}</span>
        {:else}
          <span class="badge neutral">Loading</span>
        {/if}
      </div>
    </div>

    <h2>Standings</h2>
    {#if standings.length === 0}
      <div class="card empty">
        <p>No results yet.</p>
      </div>
    {:else}
      <div class="scroll">
        <table>
          <thead>
            <tr>
              <th class="num">#</th>
              <th class="num">Lane</th>
              {#if showNames}<th>Name</th>{/if}
              {#if showClubs}<th class="optional">Club</th>{/if}
              {#if showCountries}<th class="optional">Country</th>{/if}
              <th class="optional">Category</th>
              {#each stageNumbers as n (n)}<th class="num">S{n}</th>{/each}
              <th class="num">Total</th>
              {#if showX}<th class="num">X</th>{/if}
              {#if finished}<th>Scorecard</th>{/if}
            </tr>
          </thead>
          <tbody>
            {#each standings as s, i (s.lane)}
              <tr class:muted={s.withdrawn} class:leader={!s.withdrawn && rank(i) === 1}>
                <td class="num rank">{s.withdrawn ? "" : rank(i)}</td>
                <td class="num">{s.lane}</td>
                {#if showNames}<td
                    >{s.first_name} {s.last_name}{s.withdrawn ? " (withdrawn)" : ""}</td
                  >{/if}
                {#if showClubs}<td class="optional">{s.club}</td>{/if}
                {#if showCountries}<td class="optional">{s.country}</td>{/if}
                <td class="optional">{s.category}</td>
                {#each stageNumbers as n (n)}
                  <td class="num"><StageCell stage={s.stages?.find((st) => st.stage === n)} /></td>
                {/each}
                <td class="num total">{s.total}</td>
                {#if showX}<td class="num">{s.x_count}</td>{/if}
                {#if finished}<td
                    >{#if !s.withdrawn}<a
                        href={`/scorecard/${selected}/${s.lane}`}
                        aria-label={`Scorecard for lane ${s.lane}`}>View</a
                      >{/if}</td
                  >{/if}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      {#if stageNumbers.length > 0}
        <ul class="legend" aria-label="Stage state legend">
          <li>
            <StageCell stage={{ stage: 0, accepted: false, state: "not_started" }} prefix={false} /> Not
            started
          </li>
          <li>
            <StageCell stage={{ stage: 0, accepted: false, state: "in_progress" }} prefix={false} /> In
            progress
          </li>
          <li>
            <StageCell
              stage={{ stage: 0, accepted: false, state: "awaiting_acceptance" }}
              prefix={false}
            /> Awaiting acceptance
          </li>
          <li>
            <StageCell
              stage={{ stage: 0, accepted: true, state: "accepted", score: 118 }}
              prefix={false}
            /> Accepted
          </li>
        </ul>
      {/if}
    {/if}
  {/if}
  <p><a href="/admin">Admin</a></p>
</main>

<style>
  .picker select {
    font-size: var(--text-base);
    padding: var(--space-2) var(--space-3);
    max-width: 100%;
  }
  .picker {
    display: block;
    margin: var(--space-3) 0;
  }
  .status {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2) var(--space-4);
    flex-wrap: wrap;
    padding: var(--space-2) var(--space-4);
    margin: 0 0 var(--space-4);
  }
  .progress-info,
  .connection {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
  }
  .empty {
    text-align: center;
    padding: var(--space-6) var(--space-4);
  }
  .empty p {
    margin: var(--space-1) 0;
  }
  .rank,
  .total {
    font-weight: 700;
    font-size: var(--text-lg);
  }
  tr.leader > td,
  tr.leader:hover > td {
    background: var(--brand-tint);
    font-weight: 600;
  }
  .legend {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2) var(--space-4);
    list-style: none;
    padding: 0;
    margin: var(--space-3) 0;
    color: var(--text-muted);
    font-size: var(--text-sm);
  }
  .legend li {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }
  @media (min-width: 90rem) {
    .spectator {
      max-width: 96rem;
      font-size: var(--text-lg);
    }
    .spectator :global(th) {
      font-size: var(--text-base);
    }
    .total,
    .rank {
      font-size: var(--text-xl);
    }
  }
</style>
