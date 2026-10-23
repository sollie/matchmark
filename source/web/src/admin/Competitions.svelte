<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { navigate } from "../lib/router.svelte";
  import { statusTone } from "../lib/domain";
  import { courseLabel, loadRulebook, rulebook } from "../lib/rulebook.svelte";
  import { localDate } from "../lib/datetime";
  import type { components } from "../lib/api/schema";

  type Competition = components["schemas"]["Competition"];

  const datePattern = "\\d{4}-\\d{2}-\\d{2}";

  let competitions = $state<Competition[]>([]);
  let name = $state("");
  let date = $state(localDate(new Date()));
  let course = $state("150");
  let error = $state("");

  async function load() {
    const result = await api.GET("/api/competitions");
    if (result.data) competitions = result.data.competitions;
    else error = failure(result);
  }

  onMount(load);
  onMount(loadRulebook);

  async function create(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const result = await api.POST("/api/competitions", { body: { name, date, course } });
    if (result.data) navigate(`/admin/competitions/${result.data.id}`);
    else error = failure(result);
  }
</script>

<h1>Competitions</h1>
{#if error}<p class="error" role="alert">{error}</p>{/if}

<form class="row card" onsubmit={create}>
  <label>Name <input bind:value={name} maxlength="128" required /></label>
  <label>Date <input type="text" bind:value={date} placeholder="YYYY-MM-DD" pattern={datePattern} inputmode="numeric" size="10" required /></label>
  <label>
    Course of fire
    <select bind:value={course}>
      {#each rulebook.courses as c (c.key)}<option value={c.key}>{c.label}</option>{/each}
    </select>
  </label>
  <button type="submit" class="primary">Create competition</button>
</form>

<div class="card scroll">
<table>
  <thead><tr><th>Name</th><th>Date</th><th>Course</th><th>Status</th><th>Actions</th></tr></thead>
  <tbody>
    {#each competitions as c (c.id)}
      <tr>
        <td><a href={`/admin/competitions/${c.id}`}>{c.name}</a></td>
        <td>{c.date}</td>
        <td>{courseLabel(c.course)}</td>
        <td><span class="badge {statusTone[c.status]}">{c.status}</span></td>
        <td class="row-actions">
          {#if c.status === "finished"}
            <a class="button" href={`/api/competitions/${c.id}/results.xlsx`} download title="Results report (.xlsx)" aria-label={`Results report (.xlsx) for ${c.name}`}><svg class="icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true" focusable="false"><path d="M8 2v7.5M4.5 6.5 8 10l3.5-3.5M3 13h10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>XLSX</a>
            <a class="button" href={`/api/competitions/${c.id}/results.csv`} download title="Detailed CSV" aria-label={`Detailed CSV for ${c.name}`}><svg class="icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true" focusable="false"><path d="M8 2v7.5M4.5 6.5 8 10l3.5-3.5M3 13h10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>CSV</a>
          {/if}
        </td>
      </tr>
    {:else}
      <tr><td colspan="5" class="muted">No competitions yet.</td></tr>
    {/each}
  </tbody>
</table>
</div>
