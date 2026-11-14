<script lang="ts">
  import { matchRoute, router } from "../lib/router.svelte";
  import { club } from "../lib/club.svelte";
  import type { components } from "../lib/api/schema";
  import { api } from "../lib/api/client";
  import Button, { buttonVariants } from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import LoadingState from "../lib/ui/LoadingState.svelte";
  import EmptyState from "../lib/ui/EmptyState.svelte";

  type Summary = components["schemas"]["ScorecardSummary"];

  const params = $derived(matchRoute("/scorecard/:match/:lane", router.path));
  const base = $derived(
    params ? `/api/scorer/scorecard/${params.match}/${params.lane}?format=png` : "",
  );
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
      .GET("/api/scorer/scorecard/{matchID}/{lane}/summary", {
        params: { path: { matchID: Number(current.match), lane: Number(current.lane) } },
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

<main class="mx-auto w-full max-w-3xl px-4 py-4 print:max-w-none print:p-0">
  <h1 class="m-0 mb-3 text-xl print:hidden">Scorecard{params ? `, lane ${params.lane}` : ""}</h1>
  {#if phase === "loading"}
    <LoadingState label="Loading scorecard" />
  {:else if phase === "busy"}
    <ErrorState title="The scorecard is busy, please try again shortly.">
      {#snippet action()}
        <p class="m-0 mb-3 text-muted">
          Trying again in {retrySeconds}
          {retrySeconds === 1 ? "second" : "seconds"}.
        </p>
        <Button onclick={() => attempt++}>Try again</Button>
      {/snippet}
    </ErrorState>
  {:else if phase === "broken"}
    <ErrorState title="The scorecard could not be loaded.">
      {#snippet action()}<Button onclick={() => attempt++}>Try again</Button>{/snippet}
    </ErrorState>
  {:else if phase === "missing" || !params || !summary}
    <EmptyState
      title="This scorecard is not available."
      description="Scorecards appear once the match is finished and the lane has an entry."
    >
      {#snippet action()}<a href="/">Back to start</a>{/snippet}
    </EmptyState>
  {:else}
    <section class="rounded-lg border border-border bg-surface p-4 print:border-0 print:p-0">
      <p class="m-0 text-lg font-semibold">{summary.shooter}</p>
      {#if summary.scorers.length > 0}
        <ul class="m-0 mt-2 list-none p-0 text-muted print:text-black">
          {#each summary.scorers as item (item.segment)}
            <li>{item.segment_name} scored by {item.scorer}</li>
          {/each}
        </ul>
      {/if}
    </section>
    <a
      class={`${buttonVariants({ variant: "primary", size: "lg" })} my-4 w-full sm:w-auto print:hidden`}
      href={`${base}&download=1`}>Download PNG</a
    >
    <img
      class="block h-auto w-full max-w-240 rounded-md border border-border bg-white print:max-w-full print:border-0"
      src={`${base}&v=${club.version}-${loaded}-${attempt}`}
      alt={`Scorecard for ${summary.shooter}, lane ${params.lane}`}
      onerror={imageFailed}
    />
  {/if}
</main>
