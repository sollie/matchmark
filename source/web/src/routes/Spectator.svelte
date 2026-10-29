<script lang="ts">
  import { api } from "../lib/api/client";
  import { clockTime } from "../lib/datetime";
  import SegmentCell from "../lib/SegmentCell.svelte";
  import { openStream, type StreamState } from "../lib/sse";
  import { ScreenControl } from "../lib/screenControl.svelte";
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
  const control = new ScreenControl();

  const viewed = $derived(
    control.managed
      ? control.assignment?.state === "match"
        ? control.assignment.match_id
        : null
      : selected,
  );

  const showNames = $derived(standings.some((s) => s.last_name !== undefined));
  const showClubs = $derived(standings.some((s) => s.club !== undefined));
  const showCountries = $derived(standings.some((s) => s.country !== undefined));
  const showX = $derived(standings.some((s) => s.x_count !== undefined));
  const current = $derived(matches.find((c) => c.id === viewed));
  const finished = $derived((current?.status ?? snapshot?.status) === "finished");
  const acceptance = $derived.by(() => {
    if (!snapshot) return null;
    const segments = snapshot.standings
      .filter((s) => !s.withdrawn)
      .flatMap((s) => s.segments?.filter((st) => st.segment === snapshot?.current_segment) ?? []);
    return { accepted: segments.filter((st) => st.accepted).length, total: segments.length };
  });
  const segmentNumbers = $derived(
    [...new Set(standings.flatMap((s) => s.segments?.map((st) => st.segment) ?? []))].sort(
      (a, b) => a - b,
    ),
  );

  async function loadMatches() {
    const { data } = await api.GET("/api/spectator/matches");
    if (!data) return;
    matches = data.matches;
    if (control.managed) return;
    if (selected === null || !data.matches.some((c) => c.id === selected)) {
      selected = (data.matches.find((c) => c.status === "active") ?? data.matches[0])?.id ?? null;
    }
  }

  $effect(() => {
    void loadMatches();
  });

  $effect(() => {
    control.start();
    return () => control.stop();
  });

  $effect(() => {
    const id = viewed;
    snapshot = null;
    standings = [];
    updated = null;
    if (id === null) return;
    const stream = openStream({
      url: `/api/spectator/matches/${id}/stream`,
      onEvent: (event, data) => {
        if (event !== "snapshot") return;
        snapshot = JSON.parse(data);
        standings = snapshot?.standings ?? [];
        updated = new Date();
        const status = snapshot?.status;
        if (status) matches = matches.map((c) => (c.id === id ? { ...c, status } : c));
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

{#snippet enable()}
  <p>
    <button onclick={() => control.enable()} disabled={control.busy}>Enable remote control</button>
  </p>
  {#if control.error}<p class="error" role="alert">{control.error}</p>{/if}
{/snippet}

<main class="spectator">
  <h1>Live standings</h1>
  {#if control.notice}
    <p class="card notice" role="alert">{control.notice}</p>
  {/if}
  {#if control.managed && control.registration && !control.assignment?.approved}
    <div class="card remote" role="region" aria-label="Remote control">
      <span>Remote control code <strong class="code">{control.registration.code}</strong></span>
      <button onclick={() => control.leave()} disabled={control.busy}>Leave remote control</button>
    </div>
  {/if}
  {#if control.managed && viewed === null}
    <div class="card empty" role="status">
      <h2>Waiting for a match</h2>
      <p class="muted">This screen will show a match as soon as one is assigned.</p>
    </div>
  {:else if !control.managed && matches.length === 0}
    <div class="card empty">
      <p>No matches are live yet.</p>
      <p class="muted">Reload this page once a match has started.</p>
    </div>
    {@render enable()}
  {:else}
    {#if !control.managed}
      <label class="picker">
        Match
        <select bind:value={selected}>
          {#each matches as c (c.id)}
            <option value={c.id}>{c.name} ({c.date}, {c.status})</option>
          {/each}
        </select>
      </label>
      {@render enable()}
    {/if}
    <div class="status card" role="status">
      <div class="progress-info">
        {#if current}<span class="badge neutral">{current.status}</span>{/if}
        {#if finished}
          <strong>Finished</strong>
          <a href={`/results/${viewed}`}>View results</a>
        {:else if snapshot && acceptance}
          <strong>Segment {snapshot.current_segment} of {snapshot.total_segments}</strong>
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
              {#each segmentNumbers as n (n)}<th class="num">S{n}</th>{/each}
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
                {#each segmentNumbers as n (n)}
                  <td class="num"
                    ><SegmentCell segment={s.segments?.find((st) => st.segment === n)} /></td
                  >
                {/each}
                <td class="num total">{s.total}</td>
                {#if showX}<td class="num">{s.x_count}</td>{/if}
                {#if finished}<td
                    >{#if !s.withdrawn}<a
                        href={`/scorecard/${viewed}/${s.lane}`}
                        aria-label={`Scorecard for lane ${s.lane}`}>View</a
                      >{/if}</td
                  >{/if}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      {#if segmentNumbers.length > 0}
        <ul class="legend" aria-label="Segment state legend">
          <li>
            <SegmentCell
              segment={{ segment: 0, accepted: false, state: "not_started" }}
              prefix={false}
            /> Not started
          </li>
          <li>
            <SegmentCell
              segment={{ segment: 0, accepted: false, state: "in_progress" }}
              prefix={false}
            /> In progress
          </li>
          <li>
            <SegmentCell
              segment={{ segment: 0, accepted: false, state: "awaiting_acceptance" }}
              prefix={false}
            /> Awaiting acceptance
          </li>
          <li>
            <SegmentCell
              segment={{ segment: 0, accepted: true, state: "accepted", score: 118 }}
              prefix={false}
            /> Accepted
          </li>
        </ul>
      {/if}
    {/if}
  {/if}
  {#if control.managed && control.assignment?.approved}
    <p class="leave">
      <button onclick={() => control.leave()} disabled={control.busy}>Leave remote control</button>
    </p>
  {/if}
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
  .remote {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2) var(--space-4);
    flex-wrap: wrap;
    padding: var(--space-2) var(--space-4);
    margin: var(--space-3) 0;
  }
  .code {
    font-family: monospace;
    letter-spacing: 0.1em;
  }
  .notice {
    padding: var(--space-2) var(--space-4);
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
