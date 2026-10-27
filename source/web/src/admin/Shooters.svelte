<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import Wa1500Import from "./Wa1500Import.svelte";
  import NameFields from "./NameFields.svelte";
  import { normalizeFirstName, normalizeLastName } from "../lib/names";

  type Shooter = components["schemas"]["Shooter"];
  type ShooterInput = components["schemas"]["ShooterInput"];

  const blank = (): ShooterInput => ({ last_name: "", first_name: "", country: "NOR", club: "", waid: "" });

  let shooters = $state<Shooter[]>([]);
  let form = $state<ShooterInput>(blank());
  let editing = $state<number | null>(null);
  let error = $state("");
  let query = $state("");
  let importing = $state(false);
  let notice = $state("");
  let timer: ReturnType<typeof setTimeout> | undefined;

  async function load() {
    const q = query.trim();
    const result = await api.GET("/api/shooters", { params: { query: q ? { q, limit: 100 } : {} } });
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
    const body = { ...form, first_name: normalizeFirstName(form.first_name), last_name: normalizeLastName(form.last_name) };
    const result =
      editing === null
        ? await api.POST("/api/shooters", { body })
        : await api.PUT("/api/shooters/{shooterID}", { params: { path: { shooterID: editing } }, body });
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    cancel();
    await load();
  }

  function edit(s: Shooter) {
    const { id, ...rest } = s;
    editing = id;
    form = rest;
  }

  function cancel() {
    editing = null;
    form = blank();
  }

  async function remove(s: Shooter) {
    if (!confirm(`Delete ${s.first_name} ${s.last_name}?`)) return;
    error = "";
    const result = await api.DELETE("/api/shooters/{shooterID}", { params: { path: { shooterID: s.id } } });
    if (!result.response.ok) error = failure(result);
    else await load();
  }
</script>

<h1>Shooters</h1>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if notice}<p role="status">{notice}</p>{/if}

<div class="row">
  <label>Search shooters <input type="search" bind:value={query} oninput={onsearch} onkeydown={onenter} placeholder="Name or WAID" /></label>
  <button type="button" class="secondary" aria-expanded={importing} onclick={() => (importing = !importing)}>
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
  {#if editing !== null}<button type="button" class="secondary" onclick={cancel}>Cancel</button>{/if}
</form>

<div class="card scroll">
<table>
  <thead>
    <tr><th>Name</th><th>Country</th><th>Club</th><th>WAID</th><th></th></tr>
  </thead>
  <tbody>
    {#each shooters as s (s.id)}
      <tr>
        <td>{s.first_name} {s.last_name}</td>
        <td>{s.country}</td>
        <td>{s.club}</td>
        <td>{s.waid || "—"}</td>
        <td class="row-actions">
          <button class="secondary" onclick={() => edit(s)}>Edit</button>
          <button class="danger" onclick={() => remove(s)}>Delete</button>
        </td>
      </tr>
    {:else}
      <tr><td colspan="6" class="muted">{query.trim() ? "No matching shooters." : "No shooters yet."}</td></tr>
    {/each}
  </tbody>
</table>
</div>
