<script lang="ts">
  import { api } from "../lib/api/client";
  import { clockTime } from "../lib/datetime";
  import type { components } from "../lib/api/schema";

  type Competition = components["schemas"]["Competition"];
  type Standing = components["schemas"]["Standing"];
  type Progress = components["schemas"]["StageProgress"];

  const refreshMs = 5000;

  let competitions = $state<Competition[]>([]);
  let selected = $state<number | null>(null);
  let standings = $state<Standing[]>([]);
  let progress = $state<Progress[]>([]);
  let error = $state("");
  let updated = $state<Date | null>(null);

  const showNames = $derived(standings.some((s) => s.last_name !== undefined));
  const showClubs = $derived(standings.some((s) => s.club !== undefined));
  const showCountries = $derived(standings.some((s) => s.country !== undefined));
  const showX = $derived(standings.some((s) => s.x_count !== undefined));
  const stageNumbers = $derived(
    [...new Set(standings.flatMap((s) => s.stages?.map((st) => st.stage) ?? []))].sort(
      (a, b) => a - b,
    ),
  );

  async function loadCompetitions() {
    const { data } = await api.GET("/api/spectator/competitions");
    if (!data) return;
    competitions = data.competitions;
    if (selected === null || !data.competitions.some((c) => c.id === selected)) {
      selected = (data.competitions.find((c) => c.status === "active") ?? data.competitions[0])?.id ?? null;
    }
  }

  async function refresh() {
    if (selected === null) return;
    const path = { competitionID: selected };
    const [st, pr] = await Promise.all([
      api.GET("/api/spectator/competitions/{competitionID}/standings", { params: { path } }),
      api.GET("/api/spectator/competitions/{competitionID}/progress", { params: { path } }),
    ]);
    if (st.data && pr.data) {
      standings = st.data.standings;
      progress = pr.data.stages;
      updated = new Date();
      error = "";
    } else {
      error = "Could not refresh. Retrying…";
    }
  }

  $effect(() => {
    void loadCompetitions();
    const id = setInterval(loadCompetitions, refreshMs * 6);
    return () => clearInterval(id);
  });

  $effect(() => {
    if (selected === null) return;
    void refresh();
    const id = setInterval(refresh, refreshMs);
    return () => clearInterval(id);
  });

  function rank(i: number): number {
    let r = i;
    while (r > 0 && standings[r - 1].total === standings[i].total) r--;
    return r + 1;
  }
</script>

<main class="spectator">
  <h1>Live standings</h1>
  {#if competitions.length === 0}
    <div class="card empty">
      <p>No competitions are live yet.</p>
      <p class="muted">This page updates automatically when a competition starts.</p>
    </div>
  {:else}
    <label class="picker">
      Competition
      <select bind:value={selected}>
        {#each competitions as c (c.id)}
          <option value={c.id}>{c.name} ({c.date}, {c.status})</option>
        {/each}
      </select>
    </label>

    <p class="status" role="status">
      {#if error}
        <span class="badge danger">Error</span> <span class="error">{error}</span>
      {:else if updated}
        <span class="badge success">Live</span> <span class="muted">Updated {clockTime(updated)}</span>
      {:else}
        <span class="badge neutral">Loading</span> <span class="muted">Waiting for results…</span>
      {/if}
    </p>

    {#if progress.length > 0}
      <div class="card">
        <h2>Stage progress</h2>
        <ul class="progress">
          {#each progress as p (p.stage)}
            <li>
              <span class="stage-name">Stage {p.stage}</span>
              <progress max={p.entries} value={p.locked}></progress>
              <span class="count">{p.locked}/{p.entries} locked</span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}

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
            </tr>
          </thead>
          <tbody>
            {#each standings as s, i (s.lane)}
              <tr class:muted={s.withdrawn} class:leader={!s.withdrawn && rank(i) === 1}>
                <td class="num rank">{s.withdrawn ? "" : rank(i)}</td>
                <td class="num">{s.lane}</td>
                {#if showNames}<td>{s.first_name} {s.last_name}{s.withdrawn ? " (withdrawn)" : ""}</td>{/if}
                {#if showClubs}<td class="optional">{s.club}</td>{/if}
                {#if showCountries}<td class="optional">{s.country}</td>{/if}
                <td class="optional">{s.category}</td>
                {#each stageNumbers as n (n)}
                  <td class="num">{s.stages?.find((st) => st.stage === n)?.score ?? ""}</td>
                {/each}
                <td class="num total">{s.total}</td>
                {#if showX}<td class="num">{s.x_count}</td>{/if}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
  <p><a href="/admin">Admin</a></p>
</main>

<style>
  .picker select { font-size: var(--text-base); padding: var(--space-2) var(--space-3); max-width: 100%; }
  .status { display: flex; align-items: center; gap: var(--space-2); flex-wrap: wrap; margin: var(--space-3) 0; }
  .empty { text-align: center; padding: var(--space-6) var(--space-4); }
  .empty p { margin: var(--space-1) 0; }
  .rank, .total { font-weight: 700; font-size: var(--text-lg); }
  tr.leader > td, tr.leader:hover > td { background: var(--brand-tint); font-weight: 600; }
  .progress li { flex-wrap: wrap; }
  .stage-name { min-width: 4.5rem; font-weight: 600; }
  .count { min-width: 6.5rem; text-align: right; color: var(--text-muted); }
  progress { appearance: none; height: var(--space-3); border: 0; border-radius: 999px; overflow: hidden; background: var(--neutral-bg); }
  progress::-webkit-progress-bar { background: var(--neutral-bg); }
  progress::-webkit-progress-value { background: var(--brand); border-radius: 999px; }
  progress::-moz-progress-bar { background: var(--brand); border-radius: 999px; }
  @media (max-width: 40rem) {
    .progress progress { flex-basis: 100%; order: 3; }
  }
  @media (min-width: 90rem) {
    .spectator { max-width: 96rem; font-size: var(--text-lg); }
    .spectator :global(th) { font-size: var(--text-base); }
    .total, .rank { font-size: var(--text-xl); }
  }
</style>
