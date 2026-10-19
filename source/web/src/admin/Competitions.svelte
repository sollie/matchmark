<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { navigate } from "../lib/router.svelte";
  import { courses } from "../lib/domain";
  import type { components } from "../lib/api/schema";

  type Competition = components["schemas"]["Competition"];

  let competitions = $state<Competition[]>([]);
  let name = $state("");
  let date = $state(new Date().toISOString().slice(0, 10));
  let course = $state("150");
  let error = $state("");

  async function load() {
    const result = await api.GET("/api/competitions");
    if (result.data) competitions = result.data.competitions;
    else error = failure(result);
  }

  onMount(load);

  async function create(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const result = await api.POST("/api/competitions", { body: { name, date, course } });
    if (result.data) navigate(`/admin/competitions/${result.data.id}`);
    else error = failure(result);
  }
</script>

<h1>Competitions</h1>
{#if error}<p class="error">{error}</p>{/if}

<form class="row card" onsubmit={create}>
  <label>Name <input bind:value={name} maxlength="128" required /></label>
  <label>Date <input type="date" bind:value={date} required /></label>
  <label>
    Course of fire
    <select bind:value={course}>
      {#each courses as c (c.key)}<option value={c.key}>{c.label}</option>{/each}
    </select>
  </label>
  <button type="submit">Create competition</button>
</form>

<table>
  <thead><tr><th>Name</th><th>Date</th><th>Course</th><th>Status</th></tr></thead>
  <tbody>
    {#each competitions as c (c.id)}
      <tr>
        <td><a href={`/admin/competitions/${c.id}`}>{c.name}</a></td>
        <td>{c.date}</td>
        <td>{courses.find((x) => x.key === c.course)?.label ?? c.course}</td>
        <td>{c.status}</td>
      </tr>
    {:else}
      <tr><td colspan="4" class="muted">No competitions yet.</td></tr>
    {/each}
  </tbody>
</table>
