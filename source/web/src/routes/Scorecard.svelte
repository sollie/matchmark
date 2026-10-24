<script lang="ts">
  import { matchRoute, router } from "../lib/router.svelte";
  import { club } from "../lib/club.svelte";

  const params = $derived(matchRoute("/scorecard/:competition/:lane", router.path));
  const base = $derived(params ? `/api/scorer/scorecard/${params.competition}/${params.lane}?format=png` : "");
  const loaded = Date.now();
  let failed = $state(false);

  $effect(() => {
    void base;
    failed = false;
  });
</script>

<main class="scorecard">
  <h1>Scorecard{params ? `, lane ${params.lane}` : ""}</h1>
  {#if !params || failed}
    <div class="card">
      <p>This scorecard is not available.</p>
      <p class="muted">Scorecards appear once the competition is finished and the lane has an entry.</p>
      <p><a href="/">Back to start</a></p>
    </div>
  {:else}
    <p>
      <a class="button primary download" href={`${base}&download=1`}>Download PNG</a>
    </p>
    <img
      class="card-image"
      src={`${base}&v=${club.version}-${loaded}`}
      alt={`Scorecard for lane ${params.lane}`}
      onerror={() => (failed = true)}
    />
  {/if}
</main>

<style>
  .card-image { display: block; width: 100%; max-width: 60rem; height: auto; background: #fff; border: 1px solid var(--border, #ccc); }
  .download { display: inline-block; font-size: var(--text-lg); padding: var(--space-3) var(--space-5); }
</style>
