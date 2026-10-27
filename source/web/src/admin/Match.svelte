<script lang="ts">
  import { onMount } from "svelte";
  import { MatchPage } from "../lib/matchPage.svelte";
  import { nextStatus, statusTone } from "../lib/domain";
  import { courseLabel, loadRulebook } from "../lib/rulebook.svelte";
  import Visibility from "./Visibility.svelte";
  import ControlRoom from "./ControlRoom.svelte";
  import StageHistory from "./StageHistory.svelte";
  import Entries from "./Entries.svelte";
  import Pairings from "./Pairings.svelte";
  import MatchEdit from "./MatchEdit.svelte";

  let { id }: { id: number } = $props();

  const page = $derived(new MatchPage(id));
  let editing = $state(false);

  onMount(loadRulebook);

  $effect(() => page.start());

  async function advance() {
    const next = page.next;
    if (!next) return;
    const message =
      next === "finished"
        ? "Finish this match? All pairings are revoked."
        : "Start this match? Entries are then frozen and devices score one stage at a time.";
    if (!confirm(message)) return;
    await page.advance();
  }

  async function remove() {
    if (!page.match || !confirm(`Delete match "${page.match.name}" and all its results?`)) return;
    await page.remove();
  }
</script>

<p><a href="/admin">← Matches</a></p>
{#if page.match}
  {@const match = page.match}
  <h1>{match.name}</h1>
  <p>
    {match.date} · {courseLabel(match.course)} ·
    <span class="badge {statusTone[match.status]}">{match.status}</span>
    {#if match.status === "active"}· Stage {match.current_stage}{/if}
  </p>
  <div class="row">
    {#if nextStatus[match.status]}
      <button class="primary" onclick={advance}>
        {match.status === "draft" ? "Start match" : "Finish match"}
      </button>
    {/if}
    {#if match.status === "draft" && !editing}
      <button onclick={() => (editing = true)}>Edit match</button>
    {/if}
    <button class="danger" onclick={remove}>Delete match</button>
  </div>
  {#if page.error}<p class="error" role="alert">{page.error}</p>{/if}
  {#if editing && match.status === "draft"}
    <MatchEdit
      {match}
      hasEntries={page.entries.length > 0}
      saved={(updated) => {
        page.saved(updated);
        editing = false;
      }}
      cancel={() => (editing = false)}
    />
  {/if}

  {#if match.status === "active"}
    <ControlRoom matchId={id} version={page.version} changed={page.refresh} />
  {/if}

  {#if match.status !== "draft"}
    <StageHistory matchId={id} version={page.version} />
  {/if}

  <Entries matchId={id} course={match.course} status={match.status} entries={page.entries} shooters={page.shooters} reload={page.loadEntries} />
  <Pairings matchId={id} entries={page.entries} finished={match.status === "finished"} draft={match.status === "draft"} />
  <Visibility matchId={id} />
{:else if page.error}
  <p class="error">{page.error}</p>
{:else}
  <p class="muted">Loading…</p>
{/if}
