<script lang="ts">
  import { api } from "../lib/api/client";
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
  <h1>MatchMark</h1>
  {#if competitions.length === 0}
    <p class="muted">No competitions are live yet.</p>
  {:else}
    <label>
      Competition
      <select bind:value={selected}>
        {#each competitions as c (c.id)}
          <option value={c.id}>{c.name} ({c.date}, {c.status})</option>
        {/each}
      </select>
    </label>

    {#if progress.length > 0}
      <div class="card">
        <h2>Stage progress</h2>
        <ul class="progress">
          {#each progress as p (p.stage)}
            <li>
              <span>Stage {p.stage}</span>
              <progress max={p.entries} value={p.locked}></progress>
              <span>{p.locked}/{p.entries} locked</span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}

    <h2>Standings</h2>
    <div class="scroll">
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Lane</th>
            {#if showNames}<th>Name</th>{/if}
            {#if showClubs}<th>Club</th>{/if}
            {#if showCountries}<th>Country</th>{/if}
            <th>Category</th>
            {#each stageNumbers as n (n)}<th>S{n}</th>{/each}
            <th>Total</th>
            {#if showX}<th>X</th>{/if}
          </tr>
        </thead>
        <tbody>
          {#each standings as s, i (s.lane)}
            <tr class:muted={s.withdrawn}>
              <td>{s.withdrawn ? "" : rank(i)}</td>
              <td>{s.lane}</td>
              {#if showNames}<td>{s.last_name} {s.first_name}{s.withdrawn ? " (withdrawn)" : ""}</td>{/if}
              {#if showClubs}<td>{s.club}</td>{/if}
              {#if showCountries}<td>{s.country}</td>{/if}
              <td>{s.category}</td>
              {#each stageNumbers as n (n)}
                <td>{s.stages?.find((st) => st.stage === n)?.score ?? ""}</td>
              {/each}
              <td><strong>{s.total}</strong></td>
              {#if showX}<td>{s.x_count}</td>{/if}
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if error}<p class="error">{error}</p>{/if}
    {#if updated}<p class="muted">Updated {updated.toLocaleTimeString()}</p>{/if}
  {/if}
  <p><a href="/admin">Admin</a></p>
</main>
