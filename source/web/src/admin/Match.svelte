<script lang="ts">
  import { onMount } from "svelte";
  import { MatchPage } from "../lib/matchPage.svelte";
  import { nextStatus, statusTone } from "../lib/domain";
  import { courseLabel, loadRulebook, segmentLabel } from "../lib/rulebook.svelte";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import ConfirmDialog from "../lib/ui/ConfirmDialog.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import LoadingState from "../lib/ui/LoadingState.svelte";
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import Visibility from "./Visibility.svelte";
  import ControlRoom from "./ControlRoom.svelte";
  import SegmentHistory from "./SegmentHistory.svelte";
  import Entries from "./Entries.svelte";
  import Pairings from "./Pairings.svelte";
  import MatchEdit from "./MatchEdit.svelte";

  let { id }: { id: number } = $props();

  type Pending = "advance" | "draft" | "delete";

  const page = $derived(new MatchPage(id));
  let editing = $state(false);
  let pending = $state<Pending | null>(null);

  onMount(loadRulebook);

  $effect(() => page.start());

  const prompts = $derived.by(() => {
    const match = page.match;
    const finishing = page.next === "finished";
    return {
      advance: {
        title: finishing ? "Finish match" : "Start match",
        description: finishing
          ? "Finish this match? All pairings are revoked."
          : "Start this match? Entries are then frozen and devices score one segment at a time.",
        confirmLabel: finishing ? "Finish match" : "Start match",
        tone: finishing ? ("danger" as const) : ("primary" as const),
        run: () => page.advance(),
      },
      draft: {
        title: "Back to draft",
        description:
          "Return this match to draft? Devices will show 'not started' and spectators won't see the match until you start it again. Only possible while no hits are entered.",
        confirmLabel: "Back to draft",
        tone: "primary" as const,
        run: () => page.backToDraft(),
      },
      delete: {
        title: "Delete match",
        description: `Delete match "${match?.name ?? ""}" and all its results? This cannot be undone.`,
        confirmLabel: "Delete match",
        tone: "danger" as const,
        run: () => page.remove(),
      },
    };
  });

  const prompt = $derived(pending ? prompts[pending] : null);
</script>

<div class="flex flex-col gap-4">
  <a
    href="/admin"
    class="inline-flex min-h-11 w-fit items-center gap-1 text-link outline-offset-2 focus-visible:outline-3 focus-visible:outline-focus"
    ><ArrowLeft class="size-4" aria-hidden="true" /> Matches</a
  >
  {#if page.match}
    {@const match = page.match}
    <header class="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 shadow-card">
      <div class="flex flex-wrap items-center gap-3">
        <h1 class="m-0 text-xl">{match.name}</h1>
        <Badge tone={statusTone[match.status]}>{match.status}</Badge>
      </div>
      <p class="m-0 text-muted">
        {match.date} · {courseLabel(match.course)}
        {#if match.status === "active"}· {segmentLabel(match.course, match.current_segment)}{/if}
      </p>
      {#if match.effective_referees.length > 0}
        <p class="m-0 text-sm text-muted">
          {match.effective_referees.length > 1 ? "Referees" : "Referee"}: {match.effective_referees
            .map((o) => `${o.name} (${o.waid})`)
            .join(", ")}
        </p>
      {/if}
      {#if match.effective_range_officers.length > 0}
        <p class="m-0 text-sm text-muted">
          {match.effective_range_officers.length > 1 ? "Range officers" : "Range officer"}: {match.effective_range_officers
            .map((o) => `${o.name} (${o.waid})`)
            .join(", ")}
        </p>
      {/if}
      <div class="flex flex-wrap gap-2">
        {#if nextStatus[match.status]}
          <Button onclick={() => (pending = "advance")}>
            {match.status === "draft" ? "Start match" : "Finish match"}
          </Button>
        {/if}
        {#if page.canBackToDraft}
          <Button variant="secondary" onclick={() => (pending = "draft")}>Back to draft</Button>
        {/if}
        {#if !editing}
          <Button variant="secondary" onclick={() => (editing = true)}>Edit match</Button>
        {/if}
        <Button variant="danger" class="sm:ml-auto" onclick={() => (pending = "delete")}
          >Delete match</Button
        >
      </div>
    </header>
    {#if page.error}<ErrorState title={page.error} />{/if}
    {#if editing}
      <MatchEdit
        {match}
        hasEntries={page.entries.length > 0}
        draft={match.status === "draft"}
        saved={(updated) => {
          page.saved(updated);
          editing = false;
        }}
        cancel={() => (editing = false)}
      />
    {/if}

    {#if match.status === "active"}
      <ControlRoom
        matchId={id}
        course={match.course}
        version={page.version}
        changed={page.refresh}
      />
    {/if}

    {#if match.status !== "draft"}
      <SegmentHistory matchId={id} course={match.course} version={page.version} />
    {/if}

    <Entries
      matchId={id}
      course={match.course}
      status={match.status}
      entries={page.entries}
      shooters={page.shooters}
      reload={page.loadEntries}
    />
    <Pairings
      matchId={id}
      entries={page.entries}
      finished={match.status === "finished"}
      draft={match.status === "draft"}
    />
    <Visibility matchId={id} />
  {:else if page.error}
    <ErrorState title={page.error} />
  {:else}
    <LoadingState />
  {/if}
</div>

{#if prompt}
  <ConfirmDialog
    title={prompt.title}
    description={prompt.description}
    confirmLabel={prompt.confirmLabel}
    tone={prompt.tone}
    onconfirm={async () => {
      const run = prompt.run;
      pending = null;
      await run();
    }}
    oncancel={() => (pending = null)}
  />
{/if}
