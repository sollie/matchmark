<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";

  type Shooter = components["schemas"]["Shooter"];
  type ShooterInput = components["schemas"]["ShooterInput"];

  const blank = (): ShooterInput => ({ last_name: "", first_name: "", sex: "M", country: "NOR", club: "", waid: "" });

  let shooters = $state<Shooter[]>([]);
  let form = $state<ShooterInput>(blank());
  let editing = $state<number | null>(null);
  let error = $state("");

  async function load() {
    const result = await api.GET("/api/shooters");
    if (result.data) shooters = result.data.shooters;
    else error = failure(result);
  }

  onMount(load);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const result =
      editing === null
        ? await api.POST("/api/shooters", { body: form })
        : await api.PUT("/api/shooters/{shooterID}", { params: { path: { shooterID: editing } }, body: form });
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
{#if error}<p class="error">{error}</p>{/if}

<form class="row card" onsubmit={save}>
  <label>Last name <input bind:value={form.last_name} maxlength="64" required /></label>
  <label>First name <input bind:value={form.first_name} maxlength="64" required /></label>
  <label>
    Sex
    <select bind:value={form.sex}>
      <option value="M">M</option>
      <option value="F">F</option>
    </select>
  </label>
  <label>Country <input bind:value={form.country} maxlength="3" size="4" required /></label>
  <label>Club <input bind:value={form.club} maxlength="128" required /></label>
  <label>WAID (blank for new shooter) <input bind:value={form.waid} maxlength="32" /></label>
  <button type="submit">{editing === null ? "Add shooter" : "Save"}</button>
  {#if editing !== null}<button type="button" onclick={cancel}>Cancel</button>{/if}
</form>

<table>
  <thead>
    <tr><th>Name</th><th>Sex</th><th>Country</th><th>Club</th><th>WAID</th><th></th></tr>
  </thead>
  <tbody>
    {#each shooters as s (s.id)}
      <tr>
        <td>{s.last_name}, {s.first_name}</td>
        <td>{s.sex}</td>
        <td>{s.country}</td>
        <td>{s.club}</td>
        <td>{s.waid || "—"}</td>
        <td>
          <button onclick={() => edit(s)}>Edit</button>
          <button onclick={() => remove(s)}>Delete</button>
        </td>
      </tr>
    {:else}
      <tr><td colspan="6" class="muted">No shooters yet.</td></tr>
    {/each}
  </tbody>
</table>
