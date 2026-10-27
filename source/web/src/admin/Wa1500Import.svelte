<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import { normalizeFirstName, normalizeLastName } from "../lib/names";
  import NameFields from "./NameFields.svelte";

  type Shooter = components["schemas"]["Shooter"];
  type ShooterInput = components["schemas"]["ShooterInput"];
  type Hit = components["schemas"]["Wa1500Shooter"];

  let {
    query = "",
    onsaved,
    oncancel,
  }: {
    query?: string;
    onsaved: (shooter: Shooter) => void;
    oncancel?: () => void;
  } = $props();

  const blank = (): ShooterInput => ({ last_name: "", first_name: "", country: "", club: "", waid: "" });

  let text = $state(query);
  let hits = $state<Hit[]>([]);
  let searched = $state(false);
  let searching = $state(false);
  let error = $state("");
  let manual = $state(false);
  let form = $state<ShooterInput>(blank());
  let pending = $state<{ hit: Hit; candidates: Shooter[] } | null>(null);
  let busy = $state(false);

  const hitName = (h: Hit) => `${h.first_name} ${h.last_name}`;

  let input: HTMLInputElement | undefined;

  onMount(() => {
    input?.focus();
    if (text.trim()) runSearch();
  });

  function search(e: SubmitEvent) {
    e.preventDefault();
    runSearch();
  }

  async function runSearch() {
    const q = text.trim();
    if (!q) return;
    error = "";
    searching = true;
    const result = await api.GET("/api/wa1500/search", { params: { query: { q } } });
    searching = false;
    searched = true;
    if (result.data) {
      hits = result.data.results;
      return;
    }
    hits = [];
    error = failure(result);
    if (result.response.status === 503) manual = true;
  }

  async function finish(id: number) {
    const result = await api.GET("/api/shooters/{shooterID}", { params: { path: { shooterID: id } } });
    if (result.data) onsaved(result.data);
    else error = failure(result);
  }

  async function runImport(hit: Hit, extra: { update_shooter_id?: number; create_new?: boolean } = {}) {
    error = "";
    busy = true;
    const result = await api.POST("/api/shooters/import", { body: { ...hit, ...extra } });
    busy = false;
    if (result.response.status === 409 && result.error && "candidates" in result.error) {
      pending = { hit, candidates: result.error.candidates };
      return;
    }
    if (!result.data) {
      pending = null;
      error = failure(result);
      return;
    }
    pending = null;
    await finish(result.data.id);
  }

  async function saveManual(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    busy = true;
    const body = { ...form, first_name: normalizeFirstName(form.first_name), last_name: normalizeLastName(form.last_name) };
    const result = await api.POST("/api/shooters", { body });
    busy = false;
    if (!result.data) {
      error = failure(result);
      return;
    }
    await finish(result.data.id);
  }
</script>

<div class="wa1500-import card">
  <form class="row" onsubmit={search}>
    <label>Search WA1500 <input type="text" bind:this={input} bind:value={text} maxlength="100" placeholder="Name, club or WAID" /></label>
    <button type="submit" class="primary" disabled={searching || !text.trim()}>Search</button>
    {#if oncancel}<button type="button" class="secondary" onclick={oncancel}>Close</button>{/if}
  </form>

  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#if searching}
    <p class="muted" role="status">Searching WA1500…</p>
  {:else if searched && hits.length === 0 && !error}
    <p class="muted" role="status">No WA1500 results</p>
  {/if}

  {#if hits.length > 0}
    <ul class="hits" aria-label="WA1500 results">
      {#each hits as h (h.waid)}
        <li class="row">
          <span>{hitName(h)} · {h.waid} · {h.club} · {h.country}</span>
          <button type="button" class="primary" disabled={busy} onclick={() => runImport(h)}>
            Import {hitName(h)}
          </button>
        </li>
      {/each}
    </ul>
  {/if}

  {#if !manual}
    <button type="button" class="secondary" onclick={() => (manual = true)}>Enter manually</button>
  {:else}
    <form class="row" onsubmit={saveManual} aria-label="Manual shooter entry">
      <NameFields bind:first={form.first_name} bind:last={form.last_name} />
      <label>Country <input bind:value={form.country} maxlength="3" size="4" required /></label>
      <label>Club <input bind:value={form.club} maxlength="128" required /></label>
      <label>WAID (blank for new shooter) <input bind:value={form.waid} maxlength="32" /></label>
      <button type="submit" class="primary" disabled={busy}>Save shooter</button>
    </form>
  {/if}

  {#if pending}
    {@const current = pending}
    <div class="dialog card" role="dialog" aria-modal="true" aria-labelledby="conflict-title">
      <h3 id="conflict-title">Name already exists</h3>
      <p>
        {hitName(current.hit)} matches existing shooters. Update one of them with the WA1500 data, or create a new
        shooter.
      </p>
      <ul>
        {#each current.candidates as c (c.id)}
          <li>
            <button type="button" class="secondary" disabled={busy} onclick={() => runImport(current.hit, { update_shooter_id: c.id })}>
              Update {c.first_name} {c.last_name} · {c.waid || "no WAID"} · {c.club} · {c.country}
            </button>
          </li>
        {/each}
      </ul>
      <div class="row">
        <button type="button" class="primary" disabled={busy} onclick={() => runImport(current.hit, { create_new: true })}>
          Create new shooter
        </button>
        <button type="button" class="secondary" onclick={() => (pending = null)}>Cancel</button>
      </div>
    </div>
  {/if}
</div>

<style>
  .hits { list-style: none; margin: 0; padding: 0; }
  .dialog { border: 1px solid var(--border, #ccc); background: var(--surface, #fff); }
</style>
