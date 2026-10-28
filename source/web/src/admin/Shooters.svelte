<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import Wa1500Import from "./Wa1500Import.svelte";
  import NameFields from "./NameFields.svelte";
  import { normalizeFirstName, normalizeLastName } from "../lib/names";

  type Shooter = components["schemas"]["Shooter"];
  type ShooterListItem = components["schemas"]["ShooterListItem"];
  type ShooterMatch = components["schemas"]["ShooterDetail"]["matches"][number];
  type ShooterInput = components["schemas"]["ShooterInput"];

  const blank = (): ShooterInput => ({
    last_name: "",
    first_name: "",
    country: "NOR",
    club: "",
    waid: "",
  });

  let shooters = $state<ShooterListItem[]>([]);
  let expanded = $state<number | null>(null);
  let matches = $state<ShooterMatch[]>([]);
  let form = $state<ShooterInput>(blank());
  let editing = $state<number | null>(null);
  let error = $state("");
  let query = $state("");
  let importing = $state(false);
  let notice = $state("");
  let timer: ReturnType<typeof setTimeout> | undefined;

  async function load() {
    const q = query.trim();
    const result = await api.GET("/api/shooters", {
      params: { query: q ? { q, limit: 100, include_archived: true } : {} },
    });
    if (result.data) shooters = result.data.shooters;
    else error = failure(result);
  }

  function onenter(e: KeyboardEvent) {
    if (e.key !== "Enter") return;
    clearTimeout(timer);
    load();
  }

  function onsearch() {
    clearTimeout(timer);
    timer = setTimeout(load, 250);
  }

  async function imported(s: Shooter) {
    importing = false;
    query = "";
    notice = `Saved ${s.first_name} ${s.last_name}.`;
    await load();
  }

  onMount(load);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const body = {
      ...form,
      first_name: normalizeFirstName(form.first_name),
      last_name: normalizeLastName(form.last_name),
    };
    const result =
      editing === null
        ? await api.POST("/api/shooters", { body })
        : await api.PUT("/api/shooters/{shooterID}", {
            params: { path: { shooterID: editing } },
            body,
          });
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    cancel();
    await load();
  }

  async function toggleMatches(s: ShooterListItem) {
    if (expanded === s.id) {
      expanded = null;
      return;
    }
    const result = await api.GET("/api/shooters/{shooterID}", {
      params: { path: { shooterID: s.id } },
    });
    if (!result.data) {
      error = failure(result);
      return;
    }
    matches = result.data.matches;
    expanded = s.id;
  }

  const entriesLabel = (n: number) => `Entered in ${n} ${n === 1 ? "match" : "matches"}`;

  function edit(s: Shooter) {
    const { id, archived: _archived, ...rest } = s;
    editing = id;
    form = rest;
  }

  function cancel() {
    editing = null;
    form = blank();
  }

  async function setArchived(s: ShooterListItem, archive: boolean) {
    error = "";
    const result = await (archive
      ? api.POST("/api/shooters/{shooterID}/archive", { params: { path: { shooterID: s.id } } })
      : api.POST("/api/shooters/{shooterID}/unarchive", { params: { path: { shooterID: s.id } } }));
    if (!result.response.ok) error = failure(result);
    await load();
  }

  async function remove(s: ShooterListItem) {
    if (!confirm(`Delete ${s.first_name} ${s.last_name}?`)) return;
    error = "";
    const result = await api.DELETE("/api/shooters/{shooterID}", {
      params: { path: { shooterID: s.id } },
    });
    if (!result.response.ok) error = failure(result);
    await load();
  }
</script>

<h1>Shooters</h1>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if notice}<p role="status">{notice}</p>{/if}

<div class="row">
  <label
    >Search shooters <input
      type="search"
      bind:value={query}
      oninput={onsearch}
      onkeydown={onenter}
      placeholder="Name or WAID"
    /></label
  >
  <button
    type="button"
    class="secondary"
    aria-expanded={importing}
    onclick={() => (importing = !importing)}
  >
    Import from WA1500
  </button>
</div>

{#if importing}
  <Wa1500Import onsaved={imported} oncancel={() => (importing = false)} />
{/if}

<form class="row card" onsubmit={save}>
  <NameFields bind:first={form.first_name} bind:last={form.last_name} />
  <label>Country <input bind:value={form.country} maxlength="3" size="4" required /></label>
  <label>Club <input bind:value={form.club} maxlength="128" required /></label>
  <label>WAID (blank for new shooter) <input bind:value={form.waid} maxlength="32" /></label>
  <button type="submit" class="primary">{editing === null ? "Add shooter" : "Save"}</button>
  {#if editing !== null}<button type="button" class="secondary" onclick={cancel}>Cancel</button
    >{/if}
</form>

<div class="card scroll">
  <table>
    <thead>
      <tr><th>Name</th><th>Country</th><th>Club</th><th>WAID</th><th>Entries</th><th></th></tr>
    </thead>
    <tbody>
      {#each shooters as s (s.id)}
        <tr>
          <td>
            {s.first_name}
            {s.last_name}
            {#if s.archived}<span class="badge neutral">Archived</span>{/if}
          </td>
          <td>{s.country}</td>
          <td>{s.club}</td>
          <td>{s.waid || "—"}</td>
          <td>
            {#if s.entry_count > 0}
              <button
                type="button"
                class="secondary"
                aria-expanded={expanded === s.id}
                onclick={() => toggleMatches(s)}>{entriesLabel(s.entry_count)}</button
              >
              {#if expanded === s.id}
                <ul class="matches">
                  {#each matches as m (m.id)}
                    <li><a href={`/admin/matches/${m.id}`}>{m.name}</a>, {m.date}, {m.status}</li>
                  {/each}
                </ul>
              {/if}
            {:else}
              <span class="muted">None</span>
            {/if}
          </td>
          <td class="row-actions">
            <button class="secondary" onclick={() => edit(s)}>Edit</button>
            {#if s.archived}
              <button class="secondary" onclick={() => setArchived(s, false)}>Unarchive</button>
            {:else}
              <button class="secondary" onclick={() => setArchived(s, true)}>Archive</button>
            {/if}
            <button
              class="danger"
              disabled={s.entry_count > 0}
              title={s.entry_count > 0
                ? "Remove the shooter's entries first. Entries in finished matches cannot be removed."
                : undefined}
              onclick={() => remove(s)}>Delete</button
            >
            {#if s.entry_count > 0}
              <span class="muted">Archive or merge this shooter instead.</span>
            {/if}
          </td>
        </tr>
      {:else}
        <tr
          ><td colspan="7" class="muted"
            >{query.trim() ? "No matching shooters." : "No shooters yet."}</td
          ></tr
        >
      {/each}
    </tbody>
  </table>
</div>
