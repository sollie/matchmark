<script lang="ts">
  import { api } from "../lib/api/client";
  import { club, clubLogoUrl } from "../lib/club.svelte";
  import { matchRoute, navigate, router } from "../lib/router.svelte";
  import type { components } from "../lib/api/schema";

  type Competition = components["schemas"]["Competition"];
  type Results = components["schemas"]["Results"];

  let competitions = $state<Competition[]>([]);
  let loaded = $state(false);
  let results = $state<Results | null>(null);
  let missing = $state(false);

  const id = $derived(matchRoute("/results/:id", router.path)?.id ?? null);
  const showNames = $derived(results?.categories.some((c) => c.rows.some((r) => r.last_name !== undefined)) ?? false);
  const showClubs = $derived(results?.categories.some((c) => c.rows.some((r) => r.club !== undefined)) ?? false);
  const showCountries = $derived(results?.categories.some((c) => c.rows.some((r) => r.country !== undefined)) ?? false);
  const showStages = $derived(results?.categories.some((c) => c.rows.some((r) => r.stages.some((s) => s.score !== undefined))) ?? false);
  const showX = $derived(results?.categories.some((c) => c.rows.some((r) => r.x_count !== undefined)) ?? false);

  $effect(() => {
    void (async () => {
      const { data } = await api.GET("/api/spectator/competitions");
      competitions = (data?.competitions ?? []).filter((c) => c.status === "finished");
      loaded = true;
    })();
  });

  $effect(() => {
    results = null;
    missing = false;
    if (id === null) return;
    const current = id;
    void (async () => {
      const { data, response } = await api.GET("/api/spectator/competitions/{competitionID}/results", {
        params: { path: { competitionID: Number(current) } },
      });
      if (current !== id) return;
      if (data) results = data;
      else missing = response.status === 404;
    })();
  });

  function pick(event: Event) {
    const value = (event.currentTarget as HTMLSelectElement).value;
    if (value) navigate(`/results/${value}`);
  }
</script>

<main class="results">
  <div class="no-print">
    <h1>Results</h1>
    <label class="picker">
      Finished competition
      <select value={id ?? ""} onchange={pick}>
        <option value="" disabled>{loaded && competitions.length === 0 ? "No finished competitions" : "Select a competition"}</option>
        {#each competitions as c (c.id)}
          <option value={String(c.id)}>{c.name} ({c.date})</option>
        {/each}
      </select>
    </label>
    {#if results}<button type="button" onclick={() => window.print()}>Print</button>{/if}
  </div>

  {#if missing}
    <div class="card empty"><p>Results are not available for this competition.</p></div>
  {:else if results}
    <article class="sheet">
      <header class="sheet-header">
        {#if club.hasLogo}<img class="club-logo" src={clubLogoUrl(club.version)} alt="" />{/if}
        <div>
          {#if club.name}<p class="club-name">{club.name}</p>{/if}
          <h2>{results.competition.name}</h2>
          <p class="meta">{results.competition.date} · {results.competition.course_label}</p>
        </div>
      </header>

      {#each results.categories as category (category.category)}
        <section class="category">
          <h3>{category.category}</h3>
          <div class="scroll">
            <table>
              <thead>
                <tr>
                  <th class="num">#</th>
                  {#if showNames}<th>Shooter</th><th>WAID</th>{/if}
                  {#if showClubs}<th>Club</th>{/if}
                  {#if showCountries}<th>Country</th>{/if}
                  {#if showStages}{#each results.stages as n (n)}<th class="num">S{n}</th>{/each}{/if}
                  <th class="num">Total</th>
                  {#if showX}<th class="num">X</th>{/if}
                </tr>
              </thead>
              <tbody>
                {#each category.rows as row, i (i)}
                  <tr>
                    <td class="num">{row.rank}</td>
                    {#if showNames}<td>{row.first_name} {row.last_name}</td><td>{row.waid ?? ""}</td>{/if}
                    {#if showClubs}<td>{row.club}</td>{/if}
                    {#if showCountries}<td>{row.country}</td>{/if}
                    {#if showStages}{#each row.stages as st (st.stage)}<td class="num">{st.score ?? ""}</td>{/each}{/if}
                    <td class="num total">{row.total}</td>
                    {#if showX}<td class="num">{row.x_count}</td>{/if}
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </section>
      {/each}

      {#if results.categories.length === 0}
        <p class="muted">No results.</p>
      {/if}

      {#if results.new_shooters.length > 0}
        <section class="category">
          <h3>New shooters</h3>
          <div class="scroll">
            <table>
              <thead>
                <tr>
                  <th>Shooter</th>
                  <th>Category</th>
                  {#if showClubs}<th>Club</th>{/if}
                  {#if showCountries}<th>Country</th>{/if}
                  <th class="num">Total</th>
                  {#if showX}<th class="num">X</th>{/if}
                </tr>
              </thead>
              <tbody>
                {#each results.new_shooters as s, i (i)}
                  <tr>
                    <td>{s.first_name} {s.last_name}</td>
                    <td>{s.category}</td>
                    {#if showClubs}<td>{s.club}</td>{/if}
                    {#if showCountries}<td>{s.country}</td>{/if}
                    <td class="num total">{s.total}</td>
                    {#if showX}<td class="num">{s.x_count}</td>{/if}
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </section>
      {/if}
    </article>
  {/if}
  <p class="no-print"><a href="/">Live standings</a></p>
</main>

<style>
  .picker { display: block; margin: var(--space-3) 0; }
  .picker select { font-size: var(--text-base); padding: var(--space-2) var(--space-3); max-width: 100%; }
  .empty { text-align: center; padding: var(--space-6) var(--space-4); }
  .sheet-header { display: flex; align-items: center; gap: var(--space-4); margin: var(--space-4) 0; }
  .club-logo { display: block; height: 4rem; width: auto; max-width: 8rem; object-fit: contain; }
  .club-name { margin: 0; font-weight: 600; }
  .sheet-header h2 { margin: 0; }
  .meta { margin: 0; color: var(--text-muted); }
  .category { margin: var(--space-4) 0; break-inside: avoid; }
  .total { font-weight: 700; }
  @media print {
    :global(.site-header), :global(.site-footer), .no-print { display: none !important; }
    .results { max-width: none; padding: 0; }
    .meta { color: #000; }
    table { font-size: 10pt; }
  }
</style>
