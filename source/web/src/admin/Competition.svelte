<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { courses, nextStatus } from "../lib/domain";
  import { navigate } from "../lib/router.svelte";
  import type { components } from "../lib/api/schema";
  import Visibility from "./Visibility.svelte";
  import Entries from "./Entries.svelte";
  import Pairings from "./Pairings.svelte";

  type Competition = components["schemas"]["Competition"];
  type Shooter = components["schemas"]["Shooter"];
  type EntryWithStages = components["schemas"]["EntryWithStages"];

  let { id }: { id: number } = $props();

  let competition = $state<Competition | null>(null);
  let shooters = $state<Shooter[]>([]);
  let entries = $state<EntryWithStages[]>([]);
  let error = $state("");

  async function loadCompetition() {
    const result = await api.GET("/api/competitions/{competitionID}", { params: { path: { competitionID: id } } });
    if (result.data) competition = result.data;
    else error = failure(result);
  }

  async function loadEntries() {
    const result = await api.GET("/api/competitions/{competitionID}/entries", {
      params: { path: { competitionID: id } },
    });
    if (result.data) entries = result.data.entries;
    else error = failure(result);
  }

  async function loadShooters() {
    const result = await api.GET("/api/shooters");
    if (result.data) shooters = result.data.shooters;
    else error = failure(result);
  }

  onMount(() => {
    loadCompetition();
    loadShooters();
    loadEntries();
    const timer = setInterval(loadEntries, 3000);
    return () => clearInterval(timer);
  });

  async function advance() {
    if (!competition) return;
    const next = nextStatus[competition.status];
    if (!next) return;
    const message =
      next === "finished"
        ? "Finish this competition? All pairings are revoked."
        : "Start this competition? Entries can then only change between stages.";
    if (!confirm(message)) return;
    error = "";
    const result = await api.POST("/api/competitions/{competitionID}/status", {
      params: { path: { competitionID: id } },
      body: { status: next },
    });
    if (result.data) competition = { ...competition, status: result.data.status };
    else error = failure(result);
  }

  async function remove() {
    if (!competition || !confirm(`Delete competition "${competition.name}" and all its results?`)) return;
    const result = await api.DELETE("/api/competitions/{competitionID}", { params: { path: { competitionID: id } } });
    if (result.response.ok) navigate("/admin");
    else error = failure(result);
  }
</script>

<p><a href="/admin">← Competitions</a></p>
{#if competition}
  <h1>{competition.name}</h1>
  <p>
    {competition.date} · {courses.find((c) => c.key === competition?.course)?.label ?? competition.course} ·
    <strong>{competition.status}</strong>
  </p>
  <div class="row">
    {#if nextStatus[competition.status]}
      <button onclick={advance}>
        {competition.status === "draft" ? "Start competition" : "Finish competition"}
      </button>
    {/if}
    <a href={`/api/competitions/${id}/results.xlsx`} download>Results report (.xlsx)</a>
    <a href={`/api/competitions/${id}/results.csv`} download>Detailed CSV</a>
    <button onclick={remove}>Delete competition</button>
  </div>
  {#if error}<p class="error">{error}</p>{/if}

  <Entries competitionId={id} {entries} {shooters} reload={loadEntries} />
  <Pairings competitionId={id} {entries} finished={competition.status === "finished"} />
  <Visibility competitionId={id} />
{:else if error}
  <p class="error">{error}</p>
{:else}
  <p class="muted">Loading…</p>
{/if}
