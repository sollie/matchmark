<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { navigate } from "../lib/router.svelte";
  import { statusTone } from "../lib/domain";
  import { courseLabel, loadRulebook, rulebook } from "../lib/rulebook.svelte";
  import { localDate } from "../lib/datetime";
  import CompetitionForm from "./CompetitionForm.svelte";
  import type { components } from "../lib/api/schema";

  type Match = components["schemas"]["Match"];
  type Competition = components["schemas"]["Competition"];

  const datePattern = "\\d{4}-\\d{2}-\\d{2}";

  const sameDay = "";
  const newCompetition = "new";

  let matches = $state<Match[]>([]);
  let competitions = $state<Competition[]>([]);
  let competitionChoice = $state(sameDay);
  let compName = $state("");
  let compStart = $state(localDate(new Date()));
  let compEnd = $state(localDate(new Date()));
  let creatingCompetition = $state(false);
  let editingCompetition = $state<number | null>(null);
  let name = $state("");
  let date = $state(localDate(new Date()));
  let course = $state("150");
  let error = $state("");

  async function load() {
    const [m, c] = await Promise.all([api.GET("/api/matches"), api.GET("/api/competitions")]);
    if (m.data) matches = m.data.matches;
    else error = failure(m);
    if (c.data) competitions = c.data.competitions;
    else error = failure(c);
  }

  const groups = $derived(
    competitions.map((comp) => ({ comp, matches: matches.filter((m) => m.competition_id === comp.id) })),
  );

  async function competitionSaved() {
    creatingCompetition = false;
    editingCompetition = null;
    await load();
  }

  async function removeCompetition(comp: Competition) {
    if (!confirm(`Delete competition "${comp.name}"?`)) return;
    error = "";
    const result = await api.DELETE("/api/competitions/{competitionID}", { params: { path: { competitionID: comp.id } } });
    if (result.error) error = failure(result);
    else await load();
  }

  onMount(load);
  onMount(loadRulebook);

  async function create(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const body: components["schemas"]["MatchInput"] = { name, date, course };
    if (competitionChoice === newCompetition) {
      body.competition = { name: compName, start_date: compStart, end_date: compEnd };
    } else if (competitionChoice !== sameDay) {
      body.competition_id = Number(competitionChoice);
    }
    const result = await api.POST("/api/matches", { body });
    if (result.data) navigate(`/admin/matches/${result.data.id}`);
    else error = failure(result);
  }
</script>

<h1>Matches</h1>
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
  <label>
    Competition
    <select bind:value={competitionChoice}>
      <option value={sameDay}>New one-day competition</option>
      {#each competitions as comp (comp.id)}<option value={String(comp.id)}>{comp.name} ({comp.start_date} – {comp.end_date})</option>{/each}
      <option value={newCompetition}>New competition…</option>
    </select>
  </label>
  {#if competitionChoice === newCompetition}
    <label>Competition name <input bind:value={compName} maxlength="128" required /></label>
    <label>Start date <input type="text" bind:value={compStart} placeholder="YYYY-MM-DD" pattern={datePattern} inputmode="numeric" size="10" required /></label>
    <label>End date <input type="text" bind:value={compEnd} placeholder="YYYY-MM-DD" pattern={datePattern} inputmode="numeric" size="10" required /></label>
  {/if}
  <button type="submit" class="primary">Create match</button>
</form>

<div class="row">
  <h2>Competitions</h2>
  {#if !creatingCompetition}<button type="button" onclick={() => (creatingCompetition = true)}>New competition</button>{/if}
</div>
{#if creatingCompetition}
  <CompetitionForm saved={competitionSaved} cancel={() => (creatingCompetition = false)} />
{/if}

{#each groups as { comp, matches: inComp } (comp.id)}
  <section class="competition" aria-label={`Competition ${comp.name}`}>
    <div class="row">
      <h3>{comp.name}</h3>
      <span class="muted">{comp.start_date === comp.end_date ? comp.start_date : `${comp.start_date} – ${comp.end_date}`}</span>
      <button type="button" onclick={() => (editingCompetition = comp.id)} aria-label={`Edit competition ${comp.name}`}>Edit</button>
      {#if inComp.length === 0}
        <button type="button" onclick={() => removeCompetition(comp)} aria-label={`Delete competition ${comp.name}`}>Delete</button>
      {/if}
    </div>
    {#if editingCompetition === comp.id}
      <CompetitionForm competition={comp} saved={competitionSaved} cancel={() => (editingCompetition = null)} />
    {/if}
    <div class="card scroll">
<table>
  <thead><tr><th>Name</th><th>Date</th><th>Course</th><th>Status</th><th>Actions</th></tr></thead>
  <tbody>
    {#each inComp as c (c.id)}
      <tr>
        <td><a href={`/admin/matches/${c.id}`}>{c.name}</a></td>
        <td>{c.date}</td>
        <td>{courseLabel(c.course)}</td>
        <td><span class="badge {statusTone[c.status]}">{c.status}</span></td>
        <td class="row-actions">
          {#if c.status === "finished"}
            <a class="button" href={`/api/matches/${c.id}/results.xlsx`} download title="Results report (.xlsx)" aria-label={`Results report (.xlsx) for ${c.name}`}><svg class="icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true" focusable="false"><path d="M8 2v7.5M4.5 6.5 8 10l3.5-3.5M3 13h10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>XLSX</a>
            <a class="button" href={`/api/matches/${c.id}/results.csv`} download title="Detailed CSV" aria-label={`Detailed CSV for ${c.name}`}><svg class="icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true" focusable="false"><path d="M8 2v7.5M4.5 6.5 8 10l3.5-3.5M3 13h10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>CSV</a>
            <a class="button" href={`/api/admin/matches/${c.id}/results?format=pdf`} download title="Results PDF" aria-label={`Results PDF for ${c.name}`}><svg class="icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true" focusable="false"><path d="M8 2v7.5M4.5 6.5 8 10l3.5-3.5M3 13h10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>PDF</a>
          {/if}
        </td>
      </tr>
    {:else}
      <tr><td colspan="5" class="muted">No matches yet.</td></tr>
    {/each}
  </tbody>
</table>
</div>
  </section>
{:else}
  <p class="muted">No competitions yet.</p>
{/each}
