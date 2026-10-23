<script lang="ts">
  import { api, deviceHeaders } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
  import { openStream, type StreamState } from "../lib/sse";
  import { forgetDeviceToken, loadDeviceToken } from "../lib/pairing";
  import type { components } from "../lib/api/schema";

  type Session = components["schemas"]["ScorerSession"];
  type Rings = components["schemas"]["RingCounts"];

  const rings = [
    { key: "x", label: "X" },
    { key: "10", label: "10" },
    { key: "9", label: "9" },
    { key: "8", label: "8" },
    { key: "7", label: "7" },
    { key: "0", label: "0" },
    { key: "miss", label: "Miss" },
  ] as const;

  const emptyCounts = (): Record<string, number> =>
    Object.fromEntries(rings.map((r) => [r.key, 0]));

  type SessionStage = Session["stages"][number];

  const savedCounts = (s: SessionStage | undefined): Record<string, number> =>
    s ? Object.fromEntries(rings.map((r) => [r.key, s[r.key]])) : emptyCounts();

  let token = $state(loadDeviceToken());
  let session = $state<Session | null>(null);
  let loadError = $state("");
  let counts = $state<Record<string, number>>(emptyCounts());
  let saved = $state<Record<string, number>>(emptyCounts());
  let message = $state("");
  let error = $state("");
  let busy = $state(false);
  let notice = $state("");
  let connection = $state<StreamState>("connecting");
  let connectAttempt = $state(0);
  let reviewing = $state(false);
  let started = $state(false);
  let confirmingDispute = $state(false);

  const stage = $derived(session?.current_stage ?? null);
  const current = $derived(session?.stages.find((s) => s.stage_number === stage) ?? null);
  const waitingStage = $derived(session?.stages.find((s) => s.stage_number === stage && s.status !== "accepted") ?? null);
  const acceptance = $derived(session?.acceptance ?? null);
  const maxRing = $derived(acceptance ? Math.max(1, ...rings.map((r) => acceptance[r.key])) : 1);
  const announcement = $derived(
    !session
      ? ""
      : session.phase === "finished"
        ? "Status: finished."
        : session.phase === "next_stage"
          ? `Status: stage ${session.current_stage} started.`
          : session.phase === "needs_acceptance"
            ? `Status: action needed, review your result.`
            : session.phase === "waiting"
              ? `Status: waiting.`
              : `Status: scoring lane ${session.score_lane}.`,
  );
  const sum = $derived(rings.reduce((n, r) => n + counts[r.key], 0));
  const unsaved = $derived(rings.some((r) => counts[r.key] !== saved[r.key]));
  const savedValid = $derived(
    current !== null && rings.reduce((n, r) => n + saved[r.key], 0) === current.expected_shots,
  );
  const canLock = $derived(!unsaved && savedValid);
  const saveStatus = $derived(
    unsaved
      ? "Unsaved changes: press Save, then Lock stage."
      : savedValid
        ? "Saved ✓"
        : `Enter ring counts that add up to ${current?.expected_shots} shots, then press Save.`,
  );

  const finishedScreenMs = 8000;

  function auth() {
    return deviceHeaders(token ?? "");
  }

  async function load() {
    if (!token) return;
    const { data, error: err, response } = await api.GET("/api/scorer/session", {
      headers: auth(),
    });
    if (data) {
      apply(data);
      loadError = "";
    } else if (response.status === 401 || response.status === 410) {
      forget();
    } else {
      loadError = errorMessage(err, `Request failed (${response.status})`);
    }
  }

  function forget() {
    forgetDeviceToken();
    token = null;
    session = null;
  }

  function apply(next: Session) {
    const previous = session;
    session = next;
    if (previous && previous.score_lane !== next.score_lane) {
      counts = emptyCounts();
      message = "";
      error = "";
      notice = `Score lane changed to lane ${next.score_lane}. Your unsaved ring counts were reset.`;
    }
    const now = next.stages.find((s) => s.stage_number === next.current_stage);
    saved = savedCounts(now);
    if (!previous || previous.current_stage !== next.current_stage) {
      counts = savedCounts(now);
      message = "";
      error = "";
    } else {
      const was = previous.stages.find((s) => s.stage_number === next.current_stage);
      if (now?.disputed && !was?.disputed) counts = savedCounts(now);
    }
    if (next.phase !== "next_stage") started = false;
    if (next.phase !== "needs_acceptance") {
      reviewing = false;
      confirmingDispute = false;
    }
  }

  function reconnect() {
    loadError = "";
    connectAttempt++;
  }

  $effect(() => {
    void connectAttempt;
    if (!token) return;
    let disposed = false;
    let finishTimer: ReturnType<typeof setTimeout> | undefined;
    const stream = openStream({
      url: "/api/scorer/stream",
      headers: auth(),
      onEvent: (event, data) => {
        if (event === "snapshot") {
          loadError = "";
          const next: Session = JSON.parse(data);
          apply(next);
          if (next.phase === "finished") {
            stream.close();
            finishTimer = setTimeout(forget, finishedScreenMs);
          }
        } else if (event === "revoked") {
          forget();
        }
      },
      onState: (next) => {
        connection = next;
        if (next === "ended" && !disposed && session?.phase !== "finished") void load();
      },
    });
    return () => {
      disposed = true;
      clearTimeout(finishTimer);
      stream.close();
    };
  });

  function bump(key: string, delta: number) {
    counts[key] = Math.max(0, counts[key] + delta);
  }

  async function save() {
    if (stage === null) return;
    busy = true;
    notice = "";
    message = "";
    error = "";
    const { error: err, response } = await api.PUT("/api/scorer/stages/{stage}", {
      params: { path: { stage } },
      headers: auth(),
      body: {
        x: counts.x,
        10: counts["10"],
        9: counts["9"],
        8: counts["8"],
        7: counts["7"],
        0: counts["0"],
        miss: counts.miss,
      } as Rings,
    });
    busy = false;
    if (response.ok) saved = { ...counts };
    else if (response.status === 422)
      error = `Ring counts must add up to ${current?.expected_shots} shots (entered ${sum}).`;
    else error = errorMessage(err, `Request failed (${response.status})`);
  }

  async function lock() {
    if (stage === null) return;
    busy = true;
    notice = "";
    message = "";
    error = "";
    const { error: err, response } = await api.POST("/api/scorer/stages/{stage}/lock", {
      params: { path: { stage } },
      headers: auth(),
    });
    busy = false;
    if (response.ok) {
      message = "Stage locked";
    } else if (response.status === 422) {
      error = "Save valid ring counts before locking the stage.";
    } else {
      error = errorMessage(err, `Request failed (${response.status})`);
    }
  }

  async function respond(action: "accept" | "dispute") {
    if (!acceptance) return;
    busy = true;
    error = "";
    const { error: err, response } = await api.POST(
      action === "accept" ? "/api/scorer/stages/{stage}/accept" : "/api/scorer/stages/{stage}/dispute",
      { params: { path: { stage: acceptance.stage_number } }, headers: auth() },
    );
    busy = false;
    confirmingDispute = false;
    if (response.ok) {
      reviewing = false;
      message = action === "accept" ? "Result accepted" : "Result disputed";
    } else {
      error = errorMessage(err, `Request failed (${response.status})`);
    }
  }

  function unpair() {
    forget();
  }
</script>

<main class="scorer">
  <div class="sr-only" aria-live="polite">{announcement}</div>
  {#if !token}
    <h1>Scorer</h1>
    <p>Open a pairing link from the admin to score a lane.</p>
  {:else if !session}
    <h1>Scorer</h1>
    {#if loadError}
      <p class="error" role="alert">{loadError}</p>
      <button onclick={reconnect}>Retry</button>
      <button onclick={unpair}>Unpair device</button>
    {:else}
      <p class="muted">Loading…</p>
    {/if}
  {:else}
    {#if session.phase === "finished"}
      <h1>Scorer</h1>
      <section class="card finished" aria-label="Finished">
        <h2>Finished</h2>
        <p class="banner">The competition is finished. Thank you for scoring.</p>
      </section>
    {:else}
    <h1>Scoring lane {session.score_lane}: {session.shooter_name}</h1>
    <p class="muted">This device is paired to lane {session.lane}. Stage {session.current_stage}.</p>

    {#if session.phase === "next_stage" && !started}
      <section class="card next-stage" aria-label="Next stage">
        <h2>Stage {session.current_stage} has started</h2>
        <p class="banner">The next stage has started. Get ready to score lane {session.score_lane}.</p>
        <button class="primary" onclick={() => (started = true)}>Continue</button>
      </section>
    {:else if session.phase === "scoring" || session.phase === "next_stage"}

      {#if current?.status === "open" && current.disputed}
        <p class="card banner disputed">
          Disputed: the shooter disputed stage {current.stage_number}. Correct the ring counts and lock it again.
        </p>
      {/if}

      {#if current}
        {#if current.status !== "open"}
          <p class="card locked banner">🔒 Stage {current.stage_number} is locked.</p>
        {:else}
          <div class="layout">
            <div class="rings">
              {#each rings as r (r.key)}
                <div class="ring">
                  <span class="label">{r.label}</span>
                  <button onclick={() => bump(r.key, -1)} aria-label="Decrease {r.label}">−</button>
                  <input
                    type="number"
                    inputmode="numeric"
                    min="0"
                    bind:value={counts[r.key]}
                    aria-label="{r.label} count"
                  />
                  <button onclick={() => bump(r.key, 1)} aria-label="Increase {r.label}">+</button>
                </div>
              {/each}
            </div>
            <div class="summary card">
              <p class="total" class:mismatch={sum !== current.expected_shots}>
                <strong>{sum}</strong> / {current.expected_shots}
              </p>
              <p class="total-label">
                {#if sum === current.expected_shots}
                  <span class="badge success">✓ Shots match</span>
                {:else if sum > current.expected_shots}
                  <span class="badge danger">⚠ {sum - current.expected_shots} too many</span>
                {:else}
                  <span class="badge warning">⚠ {current.expected_shots - sum} to go</span>
                {/if}
              </p>
              <div class="actions">
                <button class="primary" class:emphasis={unsaved} disabled={busy} onclick={save}>Save</button>
                <button
                  class="primary lock"
                  class:secondary={!canLock}
                  disabled={busy || !canLock}
                  aria-describedby={canLock ? undefined : "save-status"}
                  title={canLock ? undefined : saveStatus}
                  onclick={lock}>Lock stage</button>
              </div>
              <p id="save-status" class="save-status" class:unsaved aria-live="polite">{saveStatus}</p>
            </div>
          </div>
        {/if}
      {/if}
    {:else if reviewing && acceptance}
      <section class="card result" aria-label="Your result">
        <div class="hero">
          <span class="chip">Stage {acceptance.stage_number} of {session.stages.length}</span>
          <h2>Your result for stage {acceptance.stage_number}</h2>
          <p class="total">{acceptance.total} hits, {acceptance.x} X</p>
          <p class="muted">Scored by {acceptance.scorer_name}</p>
        </div>
        <ul class="breakdown">
          {#each rings as r (r.key)}
            <li>
              <span class="ring-label">{r.label}</span>
              <span class="ring-count">{acceptance[r.key]}</span>
              <span class="bar" aria-hidden="true"><span style="width: {(acceptance[r.key] / maxRing) * 100}%"></span></span>
            </li>
          {/each}
        </ul>
        <div class="action-bar">
          {#if confirmingDispute}
            <p class="confirm-text" role="alert">Dispute this result? The scorer will re-score the stage.</p>
            <div class="actions">
              <button disabled={busy} onclick={() => (confirmingDispute = false)}>Cancel</button>
              <button class="danger" disabled={busy} onclick={() => respond("dispute")}>Confirm dispute</button>
            </div>
          {:else}
            <button class="primary accept" disabled={busy} onclick={() => respond("accept")}>Accept</button>
            <button class="dispute" disabled={busy} onclick={() => (confirmingDispute = true)}>Dispute</button>
          {/if}
        </div>
      </section>
    {:else}
      <section class="card waiting" aria-label="Waiting">
        <h2>Waiting</h2>
        {#if waitingStage}
          <p class="banner">🔒 Stage {waitingStage.stage_number} is locked. Waiting for {session.shooter_name} to accept.</p>
        {:else}
          <p class="banner">🔒 Stage {session.current_stage} is locked. Waiting for the other lanes and the admin.</p>
        {/if}
        {#if acceptance}
          <div class="attention" role="group" aria-label="Action needed">
            <span class="attention-icon" aria-hidden="true">!</span>
            <p class="banner needs-acceptance">Your result for stage {acceptance.stage_number} awaits your acceptance.</p>
            <button class="primary" onclick={() => (reviewing = true)}>Review result</button>
          </div>
        {/if}
      </section>
    {/if}
    {/if}

    {#if connection === "stale"}<p class="muted">Connection lost. Reconnecting…</p>{/if}
    {#if notice}<p role="status">{notice}</p>{/if}
    {#if message}<p role="status">{message}</p>{/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <p><button onclick={unpair}>Unpair device</button></p>
  {/if}
</main>

<style>
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
  .summary { margin: 0; text-align: center; }
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
    .summary { position: sticky; top: var(--space-4); }
    .ring { grid-template-columns: 4.5rem 1fr 7rem 1fr; gap: var(--space-3); }
    .rings { gap: var(--space-3); }
  }
</style>
