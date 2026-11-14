<script lang="ts">
  import { Lock, TriangleAlert, WifiOff } from "@lucide/svelte";
  import { ScorerSession, rings } from "../lib/scorer.svelte";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import EmptyState from "../lib/ui/EmptyState.svelte";
  import LoadingState from "../lib/ui/LoadingState.svelte";
  import { toast } from "../lib/ui/toast";

  const scorer = new ScorerSession();

  $effect(() => {
    void scorer.connectAttempt;
    return scorer.start();
  });

  $effect(() => {
    if (scorer.message) toast.success(scorer.message);
  });

  $effect(() => {
    if (scorer.error) toast.error(scorer.error, { duration: 8000 });
  });

  const card = "rounded-lg border border-border bg-surface p-4 shadow-card";
  const banner = "m-0 text-lg font-semibold";
</script>

<div class="flex min-h-dvh flex-col bg-bg text-fg" style="touch-action: manipulation">
  <header class="flex min-h-14 items-center gap-3 bg-header px-4 text-header-fg">
    <img
      class="block h-8 w-auto"
      src="/brand/logo-horizontal-dark.svg"
      alt="MatchMark"
      width="158"
      height="32"
    />
    <span class="ml-auto flex items-center gap-2">
      {#if scorer.session}
        <Badge tone="neutral">Lane {scorer.session.lane}</Badge>
      {/if}
      {#if !scorer.token}
        <Badge tone="warning">Unpaired</Badge>
      {:else if scorer.connection === "stale"}
        <Badge tone="warning"><WifiOff class="size-4" aria-hidden="true" />Reconnecting</Badge>
      {:else if scorer.session}
        <Badge tone="success">Paired</Badge>
      {:else}
        <Badge tone="neutral">Pairing</Badge>
      {/if}
    </span>
  </header>

  <main class="mx-auto flex w-full max-w-xl flex-1 flex-col gap-4 p-4">
    <div class="sr-only" aria-live="polite">{scorer.announcement}</div>
    {#if !scorer.token}
      <h1 class="m-0 text-xl">Scorer</h1>
      <EmptyState
        title="Not paired"
        description="Open a pairing link from the admin to score a lane."
      />
    {:else if !scorer.session}
      <h1 class="m-0 text-xl">Scorer</h1>
      {#if scorer.loadError}
        <p
          class="m-0 rounded-lg border border-danger bg-danger-bg p-4 font-semibold text-danger-fg"
          role="alert"
        >
          {scorer.loadError}
        </p>
        <div class="flex gap-2">
          <Button onclick={() => scorer.reconnect()}>Retry</Button>
          <Button variant="secondary" onclick={() => scorer.unpair()}>Unpair device</Button>
        </div>
      {:else}
        <LoadingState />
      {/if}
    {:else}
      {#if scorer.session.phase === "not_started"}
        <h1 class="m-0 text-xl">Scorer</h1>
        <p class="m-0 text-muted">This device is paired to lane {scorer.session.lane}.</p>
        <section class={card} aria-label="Not started">
          <h2 class="m-0 mb-2 text-lg">Waiting for the match to start</h2>
          <p class={banner}>
            This device is paired. Scoring begins when the admin starts the match.
          </p>
        </section>
      {:else}
        <section
          class="flex items-center gap-4 rounded-lg border border-border bg-brand-tint p-4"
          aria-label="Score lane"
        >
          <span
            class="grid size-16 shrink-0 place-items-center rounded-lg bg-brand text-3xl font-extrabold text-on-brand tabular-nums"
            aria-hidden="true">{scorer.session.score_lane}</span
          >
          <div class="min-w-0">
            <h1 class="m-0 text-lg leading-tight">
              Scoring lane {scorer.session.score_lane}: {scorer.session.shooter_name}
            </h1>
            <p class="m-0 mt-1 text-sm text-muted">
              This device is paired to lane {scorer.session.lane}. {scorer.segmentLabel(
                scorer.session.current_segment,
              )}.
            </p>
          </div>
        </section>

        {#if scorer.session.segments.length > 1}
          <ol class="m-0 flex list-none flex-wrap gap-2 p-0" aria-label="Stages">
            {#each scorer.session.segments as seg (seg.segment_number)}
              <li>
                <Badge
                  tone={seg.status === "accepted"
                    ? "success"
                    : seg.segment_number === scorer.session.current_segment
                      ? "info"
                      : "neutral"}
                >
                  {#if seg.status !== "open"}<Lock class="size-3.5" aria-hidden="true" />{/if}
                  {seg.name ?? `Segment ${seg.segment_number}`}
                </Badge>
              </li>
            {/each}
          </ol>
        {/if}

        {#if scorer.notice}
          <p
            class="m-0 flex items-start gap-2 rounded-lg border border-warning-fg bg-warning-bg p-3 font-semibold text-warning-fg"
            role="status"
          >
            <TriangleAlert class="mt-0.5 size-5 shrink-0" aria-hidden="true" />{scorer.notice}
          </p>
        {/if}

        {#if scorer.session.phase === "next_segment" && !scorer.started}
          <section class="{card} flex flex-col gap-3" aria-label="Next segment">
            <h2 class="m-0 text-lg">
              {scorer.segmentLabel(scorer.session.current_segment)} has started
            </h2>
            <p class={banner}>
              The next segment has started. Get ready to score lane {scorer.session.score_lane}.
            </p>
            <Button size="lg" onclick={() => scorer.continueToScoring()}>Continue</Button>
          </section>
        {:else if scorer.session.phase === "scoring" || scorer.session.phase === "next_segment"}
          {#if scorer.current?.status === "open" && scorer.current.disputed}
            <p
              class="m-0 flex items-start gap-2 rounded-lg border-2 border-danger bg-danger-bg p-3 text-lg font-semibold text-danger-fg"
            >
              <TriangleAlert class="mt-0.5 size-5 shrink-0" aria-hidden="true" /><span
                >Disputed: the shooter disputed {scorer.segmentLabel(
                  scorer.current.segment_number,
                )}. Correct the ring counts and lock it again.</span
              >
            </p>
          {/if}

          {#if scorer.current}
            {#if scorer.current.status !== "open"}
              <p
                class="m-0 flex items-center gap-3 rounded-lg border border-dashed border-border bg-neutral-bg p-4 text-lg font-semibold text-neutral-fg"
              >
                🔒
                {scorer.segmentLabel(scorer.current.segment_number)} is locked.
              </p>
            {:else}
              <div class="flex flex-col gap-2">
                {#each rings as r (r.key)}
                  <div class="grid grid-cols-[3.5rem_1fr_5rem_1fr] items-center gap-2">
                    <span class="text-xl font-bold">{r.label}</span>
                    <Button
                      variant="secondary"
                      class="min-h-16 text-3xl"
                      onclick={() => scorer.bump(r.key, -1)}
                      aria-label="Decrease {r.label}">−</Button
                    >
                    <input
                      class="min-h-16 w-full min-w-0 rounded-md border border-border bg-surface text-center text-2xl font-semibold text-fg tabular-nums outline-offset-2 focus-visible:outline-3 focus-visible:outline-focus"
                      type="number"
                      inputmode="numeric"
                      min="0"
                      value={scorer.counts[r.key]}
                      oninput={(e) => {
                        const capped = scorer.setCount(r.key, e.currentTarget.valueAsNumber);
                        if (capped) e.currentTarget.value = String(scorer.counts[r.key]);
                      }}
                      aria-label="{r.label} count"
                    />
                    <Button
                      variant="secondary"
                      class="min-h-16 text-3xl"
                      disabled={scorer.atCap}
                      onclick={() => scorer.bump(r.key, 1)}
                      aria-label="Increase {r.label}">+</Button
                    >
                  </div>
                {/each}
              </div>
              <div
                class="summary sticky bottom-0 z-10 -mx-4 -mb-4 mt-auto flex flex-col gap-3 border-t border-border bg-surface p-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-overlay"
              >
                <div class="flex items-center gap-3">
                  <p
                    class={[
                      "total m-0 text-3xl leading-tight tabular-nums",
                      scorer.sum !== scorer.current.expected_shots && "text-danger-fg",
                    ]}
                  >
                    <strong>{scorer.sum}</strong> / {scorer.current.expected_shots}
                  </p>
                  <p class="m-0">
                    {#if scorer.sum === scorer.current.expected_shots}
                      <Badge tone="success">✓ Shots match</Badge>
                    {:else if scorer.sum > scorer.current.expected_shots}
                      <Badge tone="danger"
                        >⚠ {scorer.sum - scorer.current.expected_shots} too many</Badge
                      >
                    {:else}
                      <Badge tone="warning"
                        >⚠ {scorer.current.expected_shots - scorer.sum} to go</Badge
                      >
                    {/if}
                  </p>
                </div>
                {#if scorer.segmentFull}
                  <p class="m-0 text-sm font-semibold text-danger-fg" role="status">
                    Segment is full: all {scorer.current.expected_shots} shots are entered.
                  </p>
                {/if}
                <div class="flex gap-2">
                  <Button
                    size="lg"
                    class={["flex-1", scorer.unsaved && "outline-3 outline-offset-2 outline-brand"]}
                    disabled={scorer.busy}
                    onclick={() => scorer.save()}>Save</Button
                  >
                  <Button
                    size="lg"
                    variant={scorer.canLock ? "primary" : "secondary"}
                    class="flex-1"
                    disabled={scorer.busy || !scorer.canLock}
                    aria-describedby={scorer.canLock ? undefined : "save-status"}
                    title={scorer.canLock ? undefined : scorer.saveStatus}
                    onclick={() => scorer.lock()}>Lock segment</Button
                  >
                </div>
                <p
                  id="save-status"
                  class={[
                    "m-0 text-sm font-semibold",
                    scorer.unsaved ? "text-danger-fg" : "text-muted",
                  ]}
                  aria-live="polite"
                >
                  {scorer.saveStatus}
                </p>
              </div>
            {/if}
          {/if}
        {:else if scorer.reviewing && scorer.acceptance}
          <section class="{card} flex flex-col gap-4" aria-label="Your result">
            <div class="text-center">
              <Badge tone="info"
                >Segment {scorer.acceptance.segment_number} of {scorer.session.segments
                  .length}</Badge
              >
              <h2 class="m-0 my-2 text-lg">
                Your result for {scorer.segmentLabel(scorer.acceptance.segment_number)}
              </h2>
              <p class="m-0 text-4xl font-extrabold tabular-nums">
                {scorer.acceptance.total} hits, {scorer.acceptance.x} X
              </p>
              <p class="m-0 mt-1 text-muted">Scored by {scorer.acceptance.scorer_name}</p>
            </div>
            <ul class="m-0 grid list-none gap-2 p-0">
              {#each rings as r (r.key)}
                <li class="grid min-h-11 grid-cols-[3rem_3.5rem_1fr] items-center gap-3">
                  <span class="text-lg font-bold">{r.label}</span>
                  <span class="text-right text-xl font-bold tabular-nums"
                    >{scorer.acceptance[r.key]}</span
                  >
                  <span
                    class="block h-3 overflow-hidden rounded-sm border border-border bg-surface-alt"
                    aria-hidden="true"
                    ><span
                      class="block h-full bg-accent"
                      style="width: {(scorer.acceptance[r.key] / scorer.maxRing) * 100}%"
                    ></span></span
                  >
                </li>
              {/each}
            </ul>
            <div
              class="sticky bottom-0 -mx-4 -mb-4 grid gap-2 border-t border-border bg-surface p-4 shadow-overlay"
            >
              {#if scorer.confirmingDispute}
                <p class="m-0 font-semibold" role="alert">
                  Dispute this result? The scorer will re-score the segment.
                </p>
                <div class="grid grid-cols-2 gap-2">
                  <Button
                    size="lg"
                    variant="secondary"
                    disabled={scorer.busy}
                    onclick={() => scorer.cancelDispute()}>Cancel</Button
                  >
                  <Button
                    size="lg"
                    variant="danger"
                    disabled={scorer.busy}
                    onclick={() => scorer.respond("dispute")}>Confirm dispute</Button
                  >
                </div>
              {:else}
                <Button size="lg" disabled={scorer.busy} onclick={() => scorer.respond("accept")}
                  >Accept</Button
                >
                <Button
                  size="lg"
                  variant="secondary"
                  disabled={scorer.busy}
                  onclick={() => scorer.askDispute()}>Dispute</Button
                >
              {/if}
            </div>
          </section>
        {:else}
          <section class="{card} flex flex-col gap-3" aria-label="Waiting">
            <h2 class="m-0 text-lg">Waiting</h2>
            {#if scorer.waitingSegment}
              <p class="{banner} flex items-start gap-2">
                🔒
                {scorer.segmentLabel(scorer.waitingSegment.segment_number)} is locked. Waiting for {scorer
                  .session.shooter_name} to accept.
              </p>
            {:else}
              <p class="{banner} flex items-start gap-2">
                🔒
                {scorer.segmentLabel(scorer.session.current_segment)} is locked. Waiting for the other
                lanes and the admin.
              </p>
            {/if}
            {#if scorer.acceptance}
              <div
                class="grid gap-3 rounded-lg border-2 border-warning-fg bg-warning-bg p-4 text-warning-fg"
                role="group"
                aria-label="Action needed"
              >
                <p class="{banner} needs-acceptance flex items-center gap-2">
                  <TriangleAlert class="size-6 shrink-0" aria-hidden="true" />Your result for {scorer.segmentLabel(
                    scorer.acceptance.segment_number,
                  )} awaits your acceptance.
                </p>
                <Button size="lg" onclick={() => scorer.startReview()}>Review result</Button>
              </div>
            {/if}
          </section>
        {/if}
      {/if}

      {#if scorer.connection === "stale"}
        <p class="m-0 text-muted">Connection lost. Reconnecting…</p>
      {/if}
      <div class="pt-2">
        <Button variant="ghost" size="sm" onclick={() => scorer.unpair()}>Unpair device</Button>
      </div>
    {/if}
  </main>
</div>
