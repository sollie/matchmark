<script lang="ts">
  import { matchRoute, router } from "../lib/router.svelte";
  import { club } from "../lib/club.svelte";
  import type { components } from "../lib/api/schema";
  import { api } from "../lib/api/client";

  type Summary = components["schemas"]["ScorecardSummary"];

  const params = $derived(matchRoute("/scorecard/:competition/:lane", router.path));
  const base = $derived(params ? `/api/scorer/scorecard/${params.competition}/${params.lane}?format=png` : "");
  const loaded = Date.now();
  const defaultRetrySeconds = 5;
  const maxRetrySeconds = 60;
  let summary = $state<Summary | null>(null);
  let phase = $state<"loading" | "ready" | "missing" | "busy" | "broken">("loading");
  let retrySeconds = $state(defaultRetrySeconds);
  let attempt = $state(0);

  function retryAfter(response: Response): number {
    const seconds = Math.ceil(Number(response.headers.get("Retry-After")));
    if (!Number.isFinite(seconds) || seconds < 1) return defaultRetrySeconds;
    return Math.min(seconds, maxRetrySeconds);
  }

  function limited(response: Response): boolean {
    return response.status === 429 || response.status === 503;
  }

  function wait(response: Response) {
    retrySeconds = retryAfter(response);
    phase = "busy";
  }

  $effect(() => {
    const current = params;
    void attempt;
    summary = null;
    if (!current) {
      phase = "missing";
      return;
    }
    phase = "loading";
    let stale = false;
    void api
      .GET("/api/scorer/scorecard/{competitionID}/{lane}/summary", {
        params: { path: { competitionID: Number(current.competition), lane: Number(current.lane) } },
      })
      .then(({ data, response }) => {
        if (stale) return;
        if (data) {
          summary = data;
          phase = "ready";
        } else if (limited(response)) {
          wait(response);
        } else if (response.status === 404) {
          phase = "missing";
        } else {
          phase = "broken";
        }
      })
      .catch(() => {
        if (!stale) phase = "broken";
      });
    return () => {
      stale = true;
    };
  });

  $effect(() => {
    if (phase !== "busy") return;
    const timer = setTimeout(() => attempt++, retrySeconds * 1000);
    return () => clearTimeout(timer);
  });

  async function imageFailed() {
    try {
      const response = await fetch(base);
      if (response.ok) phase = "broken";
      else if (limited(response)) wait(response);
      else if (response.status === 404) phase = "missing";
      else phase = "broken";
    } catch {
      phase = "broken";
    }
  }
</script>

<main class="scorecard">
  <h1>Scorecard{params ? `, lane ${params.lane}` : ""}</h1>
  {#if phase === "loading"}
    <p class="muted">Loading scorecard…</p>
  {:else if phase === "busy"}
    <div class="card">
      <p>The scorecard is busy, please try again shortly.</p>
      <p class="muted">Trying again in {retrySeconds} {retrySeconds === 1 ? "second" : "seconds"}.</p>
      <p><button class="button" onclick={() => attempt++}>Try again</button></p>
    </div>
  {:else if phase === "broken"}
    <div class="card">
      <p>The scorecard could not be loaded.</p>
      <p><button class="button" onclick={() => attempt++}>Try again</button></p>
    </div>
  {:else if phase === "missing" || !params || !summary}
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
      src={`${base}&v=${club.version}-${loaded}-${attempt}`}
      alt={`Scorecard for ${summary.shooter}, lane ${params.lane}`}
      onerror={imageFailed}
    />
  {/if}
</main>

<style>
  .card-image { display: block; width: 100%; max-width: 60rem; height: auto; background: #fff; border: 1px solid var(--border, #ccc); }
  .download { display: inline-block; font-size: var(--text-lg); padding: var(--space-3) var(--space-5); }
</style>
