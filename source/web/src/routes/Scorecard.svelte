<script lang="ts">
  import { matchRoute, router } from "../lib/router.svelte";
  import { club } from "../lib/club.svelte";
  import type { components } from "../lib/api/schema";
  import { api } from "../lib/api/client";

  type Summary = components["schemas"]["ScorecardSummary"];

  const params = $derived(matchRoute("/scorecard/:competition/:lane", router.path));
  const base = $derived(params ? `/api/scorer/scorecard/${params.competition}/${params.lane}?format=png` : "");
  const loaded = Date.now();
  let summary = $state<Summary | null>(null);
  let failed = $state(false);
  let loading = $state(true);

  $effect(() => {
    const current = params;
    summary = null;
    failed = false;
    loading = true;
    if (!current) {
      loading = false;
      return;
    }
    let stale = false;
    void api
      .GET("/api/scorer/scorecard/{competitionID}/{lane}/summary", {
        params: { path: { competitionID: Number(current.competition), lane: Number(current.lane) } },
      })
      .then(({ data }) => {
        if (stale) return;
        summary = data ?? null;
        loading = false;
      });
    return () => {
      stale = true;
    };
  });
</script>

<main class="scorecard">
  <h1>Scorecard{params ? `, lane ${params.lane}` : ""}</h1>
  {#if loading}
    <p class="muted">Loading scorecard…</p>
  {:else if !params || !summary || failed}
    <div class="card">
      <p>This scorecard is not available.</p>
      <p class="muted">Scorecards appear once the competition is finished and the lane has an entry.</p>
      <p><a href="/">Back to start</a></p>
    </div>
  {:else}
    <section class="card">
      <p><strong>{summary.shooter}</strong></p>
      {#if summary.scorers.length > 0}
        <ul class="scorers">
          {#each summary.scorers as item (item.stage)}
            <li>Stage {item.stage} scored by {item.scorer}</li>
          {/each}
        </ul>
      {/if}
    </section>
    <p>
      <a class="button primary download" href={`${base}&download=1`}>Download PNG</a>
    </p>
    <img
      class="card-image"
      src={`${base}&v=${club.version}-${loaded}`}
      alt={`Scorecard for ${summary.shooter}, lane ${params.lane}`}
      onerror={() => (failed = true)}
    />
  {/if}
</main>

<style>
  .card-image { display: block; width: 100%; max-width: 60rem; height: auto; background: #fff; border: 1px solid var(--border, #ccc); }
  .download { display: inline-block; font-size: var(--text-lg); padding: var(--space-3) var(--space-5); }
</style>
