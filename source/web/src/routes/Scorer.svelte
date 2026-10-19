<script lang="ts">
  import { api, deviceHeaders } from "../lib/api/client";
  import { errorMessage } from "../lib/api/util";
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

  let token = $state(loadDeviceToken());
  let session = $state<Session | null>(null);
  let loadError = $state("");
  let stage = $state<number | null>(null);
  let counts = $state<Record<string, number>>(emptyCounts());
  let message = $state("");
  let error = $state("");
  let busy = $state(false);

  const current = $derived(session?.stages.find((s) => s.stage_number === stage) ?? null);
  const sum = $derived(rings.reduce((n, r) => n + counts[r.key], 0));

  function auth() {
    return deviceHeaders(token ?? "");
  }

  async function load() {
    if (!token) return;
    const { data, error: err, response } = await api.GET("/api/scorer/session", {
      headers: auth(),
    });
    if (data) {
      session = data;
      loadError = "";
      if (stage === null || !data.stages.some((s) => s.stage_number === stage)) {
        stage = (data.stages.find((s) => !s.locked) ?? data.stages[0])?.stage_number ?? null;
      }
    } else if (response.status === 401) {
      forgetDeviceToken();
      token = null;
      session = null;
    } else {
      loadError = errorMessage(err, `Request failed (${response.status})`);
    }
  }

  $effect(() => {
    void load();
  });

  function selectStage(n: number) {
    stage = n;
    counts = emptyCounts();
    message = "";
    error = "";
  }

  function bump(key: string, delta: number) {
    counts[key] = Math.max(0, counts[key] + delta);
  }

  async function save() {
    if (stage === null) return;
    busy = true;
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
    if (response.ok) message = "Saved";
    else if (response.status === 422)
      error = `Ring counts must add up to ${current?.expected_shots} shots (entered ${sum}).`;
    else error = errorMessage(err, `Request failed (${response.status})`);
  }

  async function lock() {
    if (stage === null) return;
    busy = true;
    message = "";
    error = "";
    const { error: err, response } = await api.POST("/api/scorer/stages/{stage}/lock", {
      params: { path: { stage } },
      headers: auth(),
    });
    busy = false;
    if (response.ok) {
      message = "Stage locked";
      await load();
    } else if (response.status === 422) {
      error = "Save valid ring counts before locking the stage.";
    } else {
      error = errorMessage(err, `Request failed (${response.status})`);
    }
  }

  function unpair() {
    forgetDeviceToken();
    token = null;
    session = null;
  }
</script>

<main class="scorer">
  {#if !token}
    <h1>Scorer</h1>
    <p>Open a pairing link from the admin to score a lane.</p>
  {:else if !session}
    <h1>Scorer</h1>
    {#if loadError}
      <p class="error" role="alert">{loadError}</p>
      <button onclick={load}>Retry</button>
      <button onclick={unpair}>Unpair device</button>
    {:else}
      <p class="muted">Loading…</p>
    {/if}
  {:else}
    <h1>Scoring lane {session.score_lane}: {session.shooter_name}</h1>
    <p class="muted">This device is paired to lane {session.lane}.</p>

    <div class="stages" role="tablist">
      {#each session.stages as s (s.stage_number)}
        <button
          role="tab"
          aria-selected={s.stage_number === stage}
          class:active={s.stage_number === stage}
          class:is-locked={s.locked}
          onclick={() => selectStage(s.stage_number)}
        >
          <span>Stage {s.stage_number}</span>
          <span class="state">{s.locked ? "🔒 Locked" : "Open"}</span>
        </button>
      {/each}
    </div>

    {#if current}
      {#if current.locked}
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
              <button class="primary" disabled={busy} onclick={save}>Save</button>
              <button class="primary lock" disabled={busy} onclick={lock}>Lock stage</button>
            </div>
          </div>
        </div>
      {/if}
    {/if}

    {#if message}<p role="status">{message}</p>{/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <p><button onclick={unpair}>Unpair device</button></p>
  {/if}
</main>

<style>
  .stages { display: flex; gap: var(--space-2); flex-wrap: wrap; margin: var(--space-3) 0; }
  .stages button { display: flex; flex-direction: column; align-items: center; min-width: 6rem; min-height: var(--touch); padding: var(--space-2) var(--space-4); border: 2px solid var(--border); border-radius: var(--radius); background: var(--surface); color: var(--text); font-weight: 600; }
  .stages button .state { font-size: var(--text-sm); font-weight: 500; }
  .stages button.is-locked { border-style: dashed; }
  .stages button.active { background: var(--brand); border-color: var(--brand); color: var(--on-brand); }
  .banner { font-weight: 600; font-size: var(--text-lg); }
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
  .actions .lock { background: var(--accent); border-color: var(--accent); color: var(--on-brand); }
  @media (min-width: 48rem) {
    .layout { grid-template-columns: 1fr 18rem; }
    .summary { position: sticky; top: var(--space-4); }
    .ring { grid-template-columns: 4.5rem 1fr 7rem 1fr; gap: var(--space-3); }
    .rings { gap: var(--space-3); }
  }
</style>
