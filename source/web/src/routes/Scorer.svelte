<script lang="ts">
  import { ScorerSession, rings } from "../lib/scorer.svelte";

  const scorer = new ScorerSession();

  $effect(() => {
    void scorer.connectAttempt;
    return scorer.start();
  });
</script>

<main class="scorer">
  <div class="sr-only" aria-live="polite">{scorer.announcement}</div>
  {#if !scorer.token}
    <h1>Scorer</h1>
    <p>Open a pairing link from the admin to score a lane.</p>
  {:else if !scorer.session}
    <h1>Scorer</h1>
    {#if scorer.loadError}
      <p class="error" role="alert">{scorer.loadError}</p>
      <button onclick={() => scorer.reconnect()}>Retry</button>
      <button onclick={() => scorer.unpair()}>Unpair device</button>
    {:else}
      <p class="muted">Loading…</p>
    {/if}
  {:else}
    {#if scorer.session.phase === "not_started"}
      <h1>Scorer</h1>
      <p class="muted">This device is paired to lane {scorer.session.lane}.</p>
      <section class="card" aria-label="Not started">
        <h2>Waiting for the competition to start</h2>
        <p class="banner">This device is paired. Scoring begins when the admin starts the competition.</p>
      </section>
    {:else}
    <h1>Scoring lane {scorer.session.score_lane}: {scorer.session.shooter_name}</h1>
    <p class="muted">This device is paired to lane {scorer.session.lane}. Stage {scorer.session.current_stage}.</p>

    {#if scorer.session.phase === "next_stage" && !scorer.started}
      <section class="card next-stage" aria-label="Next stage">
        <h2>Stage {scorer.session.current_stage} has started</h2>
        <p class="banner">The next stage has started. Get ready to score lane {scorer.session.score_lane}.</p>
        <button class="primary" onclick={() => scorer.continueToScoring()}>Continue</button>
      </section>
    {:else if scorer.session.phase === "scoring" || scorer.session.phase === "next_stage"}

      {#if scorer.current?.status === "open" && scorer.current.disputed}
        <p class="card banner disputed">
          Disputed: the shooter disputed stage {scorer.current.stage_number}. Correct the ring counts and lock it again.
        </p>
      {/if}

      {#if scorer.current}
        {#if scorer.current.status !== "open"}
          <p class="card locked banner">🔒 Stage {scorer.current.stage_number} is locked.</p>
        {:else}
          <div class="layout">
            <div class="rings">
              {#each rings as r (r.key)}
                <div class="ring">
                  <span class="label">{r.label}</span>
                  <button onclick={() => scorer.bump(r.key, -1)} aria-label="Decrease {r.label}">−</button>
                  <input
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
                  <button disabled={scorer.atCap} onclick={() => scorer.bump(r.key, 1)} aria-label="Increase {r.label}">+</button>
                </div>
              {/each}
            </div>
            <div class="summary card">
              <p class="total" class:mismatch={scorer.sum !== scorer.current.expected_shots}>
                <strong>{scorer.sum}</strong> / {scorer.current.expected_shots}
              </p>
              <p class="total-label">
                {#if scorer.sum === scorer.current.expected_shots}
                  <span class="badge success">✓ Shots match</span>
                {:else if scorer.sum > scorer.current.expected_shots}
                  <span class="badge danger">⚠ {scorer.sum - scorer.current.expected_shots} too many</span>
                {:else}
                  <span class="badge warning">⚠ {scorer.current.expected_shots - scorer.sum} to go</span>
                {/if}
              </p>
              {#if scorer.stageFull}
                <p class="save-status unsaved" role="status">Stage is full: all {scorer.current.expected_shots} shots are entered.</p>
              {/if}
              <div class="actions">
                <button class="primary" class:emphasis={scorer.unsaved} disabled={scorer.busy} onclick={() => scorer.save()}>Save</button>
                <button
                  class="primary lock"
                  class:secondary={!scorer.canLock}
                  disabled={scorer.busy || !scorer.canLock}
                  aria-describedby={scorer.canLock ? undefined : "save-status"}
                  title={scorer.canLock ? undefined : scorer.saveStatus}
                  onclick={() => scorer.lock()}>Lock stage</button>
              </div>
              <p id="save-status" class="save-status" class:unsaved={scorer.unsaved} aria-live="polite">{scorer.saveStatus}</p>
            </div>
          </div>
        {/if}
      {/if}
    {:else if scorer.reviewing && scorer.acceptance}
      <section class="card result" aria-label="Your result">
        <div class="hero">
          <span class="chip">Stage {scorer.acceptance.stage_number} of {scorer.session.stages.length}</span>
          <h2>Your result for stage {scorer.acceptance.stage_number}</h2>
          <p class="total">{scorer.acceptance.total} hits, {scorer.acceptance.x} X</p>
          <p class="muted">Scored by {scorer.acceptance.scorer_name}</p>
        </div>
        <ul class="breakdown">
          {#each rings as r (r.key)}
            <li>
              <span class="ring-label">{r.label}</span>
              <span class="ring-count">{scorer.acceptance[r.key]}</span>
              <span class="bar" aria-hidden="true"><span style="width: {(scorer.acceptance[r.key] / scorer.maxRing) * 100}%"></span></span>
            </li>
          {/each}
        </ul>
        <div class="action-bar">
          {#if scorer.confirmingDispute}
            <p class="confirm-text" role="alert">Dispute this result? The scorer will re-score the stage.</p>
            <div class="actions">
              <button disabled={scorer.busy} onclick={() => scorer.cancelDispute()}>Cancel</button>
              <button class="danger" disabled={scorer.busy} onclick={() => scorer.respond("dispute")}>Confirm dispute</button>
            </div>
          {:else}
            <button class="primary accept" disabled={scorer.busy} onclick={() => scorer.respond("accept")}>Accept</button>
            <button class="dispute" disabled={scorer.busy} onclick={() => scorer.askDispute()}>Dispute</button>
          {/if}
        </div>
      </section>
    {:else}
      <section class="card waiting" aria-label="Waiting">
        <h2>Waiting</h2>
        {#if scorer.waitingStage}
          <p class="banner">🔒 Stage {scorer.waitingStage.stage_number} is locked. Waiting for {scorer.session.shooter_name} to accept.</p>
        {:else}
          <p class="banner">🔒 Stage {scorer.session.current_stage} is locked. Waiting for the other lanes and the admin.</p>
        {/if}
        {#if scorer.acceptance}
          <div class="attention" role="group" aria-label="Action needed">
            <span class="attention-icon" aria-hidden="true">!</span>
            <p class="banner needs-acceptance">Your result for stage {scorer.acceptance.stage_number} awaits your acceptance.</p>
            <button class="primary" onclick={() => scorer.startReview()}>Review result</button>
          </div>
        {/if}
      </section>
    {/if}
    {/if}

    {#if scorer.connection === "stale"}<p class="muted">Connection lost. Reconnecting…</p>{/if}
    {#if scorer.notice}<p role="status">{scorer.notice}</p>{/if}
    {#if scorer.message}<p role="status">{scorer.message}</p>{/if}
    {#if scorer.error}<p class="error" role="alert">{scorer.error}</p>{/if}
    <p><button onclick={() => scorer.unpair()}>Unpair device</button></p>
  {/if}
</main>

<style>
  .scorer { touch-action: manipulation; }
  .scorer :global(button), .scorer :global(input) { touch-action: manipulation; }
  .banner { font-weight: 600; font-size: var(--text-lg); }
  .disputed { border-color: var(--danger-fg); color: var(--danger-fg); }
  .sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
  .hero { text-align: center; }
  .hero h2 { margin: var(--space-2) 0; font-size: var(--text-lg); }
  .hero .total { font-size: 2.5rem; font-weight: 800; font-variant-numeric: tabular-nums; }
  .chip { display: inline-block; padding: var(--space-1) var(--space-3); border-radius: 999px; background: var(--brand-tint); color: var(--text); font-size: var(--text-sm); font-weight: 600; }
  .breakdown { list-style: none; margin: var(--space-4) 0; padding: 0; display: grid; gap: var(--space-2); }
  .breakdown li { display: grid; grid-template-columns: 3rem 3.5rem 1fr; gap: var(--space-3); align-items: center; min-height: var(--touch); }
  .ring-label { font-weight: 700; font-size: var(--text-lg); }
  .ring-count { font-size: var(--text-xl); font-weight: 700; text-align: right; font-variant-numeric: tabular-nums; }
  .bar { display: block; height: 0.75rem; border-radius: var(--radius-sm); background: var(--surface-alt); border: 1px solid var(--border); overflow: hidden; }
  .bar span { display: block; height: 100%; background: var(--accent); }
  .action-bar { position: sticky; bottom: 0; display: grid; gap: var(--space-2); margin: 0 calc(var(--space-4) * -1) calc(var(--space-4) * -1); padding: var(--space-3) var(--space-4); background: var(--surface); border-top: 1px solid var(--border); box-shadow: var(--shadow-lg); }
  .action-bar button { min-height: 3.5rem; font-size: var(--text-lg); }
  .action-bar .danger { background: var(--danger); border-color: var(--danger); color: var(--on-danger); }
  .confirm-text { margin: 0; font-weight: 600; }
  .attention { display: grid; grid-template-columns: auto 1fr; gap: var(--space-3); align-items: center; padding: var(--space-4); border: 2px solid var(--warning-fg); border-radius: var(--radius-lg); background: var(--warning-bg); color: var(--warning-fg); }
  .attention .banner { margin: 0; }
  .attention button { grid-column: 1 / -1; min-height: 3.5rem; font-size: var(--text-lg); }
  .attention-icon { display: grid; place-items: center; width: 2.5rem; height: 2.5rem; border-radius: 50%; background: var(--warning-fg); color: var(--warning-bg); font-weight: 800; font-size: var(--text-lg); }
  @media (min-width: 48rem) {
    .action-bar { position: static; margin: var(--space-4) 0 0; padding: 0; border: 0; box-shadow: none; grid-auto-flow: column; }
  }
  .layout { display: grid; gap: var(--space-4); align-items: start; }
  .rings { display: grid; gap: var(--space-2); }
  .ring { display: grid; grid-template-columns: 3.5rem 1fr 5rem 1fr; gap: var(--space-2); align-items: center; }
  .ring .label { font-weight: 700; font-size: var(--text-lg); }
  .ring button { font-size: 1.75rem; min-height: 3.25rem; min-width: var(--touch); }
  .ring input { font-size: 1.5rem; text-align: center; min-height: 3.25rem; width: 100%; }
  .summary { margin: 0; text-align: center; position: sticky; bottom: 0; z-index: 1; padding-bottom: max(var(--space-3), env(safe-area-inset-bottom)); box-shadow: var(--shadow-lg); }
  .summary { display: grid; grid-template-columns: auto 1fr; column-gap: var(--space-3); align-items: center; }
  .summary .total { font-size: 2rem; }
  .summary .total-label { margin: 0; text-align: left; }
  .summary .actions, .summary .save-status { grid-column: 1 / -1; }
  .total { margin: 0; font-size: 2.5rem; line-height: 1.2; }
  .total.mismatch { color: var(--danger-fg); }
  .total-label { margin: var(--space-1) 0 var(--space-3); }
  .actions { display: flex; gap: var(--space-2); }
  .actions button { flex: 1; min-height: 3.5rem; font-size: var(--text-lg); }
  .actions .emphasis { box-shadow: 0 0 0 3px var(--brand-tint); }
  .actions .lock.secondary { background: var(--surface-alt); border-color: var(--border); color: var(--text); }
  .save-status { margin: var(--space-2) 0 0; font-size: var(--text-sm); font-weight: 600; }
  .save-status.unsaved { color: var(--danger-fg); }
  .actions .lock { background: var(--accent); border-color: var(--accent); color: var(--on-brand); }
  @media (min-width: 48rem) {
    .layout { grid-template-columns: 1fr 18rem; }
    .summary { top: var(--space-4); bottom: auto; display: block; padding-bottom: var(--space-4); box-shadow: none; }
    .summary .total { font-size: 2.5rem; }
    .summary .total-label { margin: var(--space-1) 0 var(--space-3); text-align: center; }
    .ring { grid-template-columns: 4.5rem 1fr 7rem 1fr; gap: var(--space-3); }
    .rings { gap: var(--space-3); }
  }
</style>
