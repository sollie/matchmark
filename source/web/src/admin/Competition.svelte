<script lang="ts">
  import { onMount } from "svelte";
  import { CompetitionPage } from "../lib/competitionPage.svelte";
  import { nextStatus, statusTone } from "../lib/domain";
  import { courseLabel, loadRulebook } from "../lib/rulebook.svelte";
  import Visibility from "./Visibility.svelte";
  import ControlRoom from "./ControlRoom.svelte";
  import StageHistory from "./StageHistory.svelte";
  import Entries from "./Entries.svelte";
  import Pairings from "./Pairings.svelte";
  import CompetitionEdit from "./CompetitionEdit.svelte";

  let { id }: { id: number } = $props();

  const page = $derived(new CompetitionPage(id));
  let editing = $state(false);

  onMount(loadRulebook);

  $effect(() => page.start());

  async function advance() {
    const next = page.next;
    if (!next) return;
    const message =
      next === "finished"
        ? "Finish this competition? All pairings are revoked."
        : "Start this competition? Entries are then frozen and devices score one stage at a time.";
    if (!confirm(message)) return;
    await page.advance();
  }

  async function remove() {
    if (!page.competition || !confirm(`Delete competition "${page.competition.name}" and all its results?`)) return;
    await page.remove();
  }
</script>

<p><a href="/admin">← Competitions</a></p>
{#if page.competition}
  {@const competition = page.competition}
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
  {#if page.error}<p class="error" role="alert">{page.error}</p>{/if}
  {#if editing && competition.status === "draft"}
    <CompetitionEdit
      {competition}
      hasEntries={page.entries.length > 0}
      saved={(updated) => {
        page.saved(updated);
        editing = false;
      }}
      cancel={() => (editing = false)}
    />
  {/if}

  {#if competition.status === "active"}
    <ControlRoom competitionId={id} version={page.version} changed={page.refresh} />
  {/if}

  {#if competition.status !== "draft"}
    <StageHistory competitionId={id} version={page.version} />
  {/if}

  <Entries competitionId={id} course={competition.course} status={competition.status} entries={page.entries} shooters={page.shooters} reload={page.loadEntries} />
  <Pairings competitionId={id} entries={page.entries} finished={competition.status === "finished"} draft={competition.status === "draft"} />
  <Visibility competitionId={id} />
{:else if page.error}
  <p class="error">{page.error}</p>
{:else}
  <p class="muted">Loading…</p>
{/if}
