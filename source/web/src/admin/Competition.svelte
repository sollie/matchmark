<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { openStream } from "../lib/sse";
  import { failure } from "../lib/api/util";
  import { nextStatus, statusTone } from "../lib/domain";
  import { courseLabel, loadRulebook } from "../lib/rulebook.svelte";
  import { navigate } from "../lib/router.svelte";
  import type { components } from "../lib/api/schema";
  import Visibility from "./Visibility.svelte";
  import ControlRoom from "./ControlRoom.svelte";
  import StageHistory from "./StageHistory.svelte";
  import Entries from "./Entries.svelte";
  import Pairings from "./Pairings.svelte";
  import CompetitionEdit from "./CompetitionEdit.svelte";

  type Competition = components["schemas"]["Competition"];
  type Shooter = components["schemas"]["Shooter"];
  type EntryWithStages = components["schemas"]["EntryWithStages"];

  let { id }: { id: number } = $props();

  let competition = $state<Competition | null>(null);
  let shooters = $state<Shooter[]>([]);
  let entries = $state<EntryWithStages[]>([]);
  let error = $state("");
  let editing = $state(false);
  let version = $state(0);

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
    loadRulebook();
    loadCompetition();
    loadShooters();
  });

  $effect(() => {
    const stream = openStream({
      url: `/api/competitions/${id}/stream`,
      onEvent: (event) => {
        if (event === "changed") {
          void loadEntries();
          void loadCompetition();
          version += 1;
        }
      },
    });
    return () => stream.close();
  });

  async function advance() {
    if (!competition) return;
    const next = nextStatus[competition.status];
    if (!next) return;
    const message =
      next === "finished"
        ? "Finish this competition? All pairings are revoked."
        : "Start this competition? Entries are then frozen and devices score one stage at a time.";
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
    {competition.date} · {courseLabel(competition.course)} ·
    <span class="badge {statusTone[competition.status]}">{competition.status}</span>
    {#if competition.status === "active"}· Stage {competition.current_stage}{/if}
  </p>
  <div class="row">
    {#if nextStatus[competition.status]}
      <button class="primary" onclick={advance}>
        {competition.status === "draft" ? "Start competition" : "Finish competition"}
      </button>
    {/if}
    {#if competition.status === "draft" && !editing}
      <button onclick={() => (editing = true)}>Edit competition</button>
    {/if}
    <button class="danger" onclick={remove}>Delete competition</button>
  </div>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if editing && competition.status === "draft"}
    <CompetitionEdit
      {competition}
      hasEntries={entries.length > 0}
      saved={(updated) => {
        competition = updated;
        editing = false;
      }}
      cancel={() => (editing = false)}
    />
  {/if}

  {#if competition.status === "active"}
    <ControlRoom competitionId={id} {version} changed={async () => { await loadCompetition(); await loadEntries(); }} />
  {/if}

  {#if competition.status !== "draft"}
    <StageHistory competitionId={id} {version} />
  {/if}

  <Entries competitionId={id} course={competition.course} status={competition.status} {entries} {shooters} reload={loadEntries} />
  <Pairings competitionId={id} {entries} finished={competition.status === "finished"} draft={competition.status === "draft"} />
  <Visibility competitionId={id} />
{:else if error}
  <p class="error">{error}</p>
{:else}
  <p class="muted">Loading…</p>
{/if}
