<script lang="ts">
  import { tick } from "svelte";
  import { api } from "../lib/api/client";
  import { settle } from "../lib/api/util";
  import { entryCommands } from "../lib/entries";
  import { courseCategories, segmentColumn, segmentLabel } from "../lib/rulebook.svelte";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import ConfirmDialog from "../lib/ui/ConfirmDialog.svelte";
  import Dialog from "../lib/ui/Dialog.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Select from "../lib/ui/Select.svelte";
  import { cn } from "../lib/ui/utils";
  import ShooterPicker from "./ShooterPicker.svelte";
  import Wa1500Import from "./Wa1500Import.svelte";
  import type { components } from "../lib/api/schema";

  type Shooter = components["schemas"]["Shooter"];
  type EntryWithSegments = components["schemas"]["EntryWithSegments"];
  type RingCounts = components["schemas"]["RingCounts"];

  let {
    matchId,
    course,
    entries,
    shooters,
    reload,
    status = "draft",
  }: {
    matchId: number;
    course: string;
    entries: EntryWithSegments[];
    shooters: Shooter[];
    reload: () => Promise<void>;
    status?: components["schemas"]["MatchStatus"];
  } = $props();

  const ringKeys = ["x", "10", "9", "8", "7", "0", "miss"] as const;
  const blankCounts = (): RingCounts => ({
    x: 0,
    "10": 0,
    "9": 0,
    "8": 0,
    "7": 0,
    "0": 0,
    miss: 0,
  });
  const numberInput =
    "min-h-11 w-full min-w-0 rounded-md border border-border bg-surface px-3 text-base text-fg outline-offset-2 hover:border-muted focus-visible:outline-3 focus-visible:outline-focus aria-invalid:border-danger";

  let shooterId = $state<number | null>(null);
  let pickerKey = $state(0);
  let importQuery = $state<string | null>(null);
  let imported = $state<Shooter[]>([]);
  let chosen = $state<Shooter | null>(null);
  let chosenCategory = $state<string | null>(null);
  let laneInput = $state<number | null>(null);
  let submitted = $state(false);
  let editing = $state<{ id: number; category: string; lane: number } | null>(null);
  let remarking = $state<{ id: number; text: string } | null>(null);
  let correcting = $state<{ id: number; classification: string; class: string } | null>(null);
  let confirming = $state<{ kind: "withdraw" | "remove"; entry: EntryWithSegments } | null>(null);
  let scoring = $state<{
    entry: EntryWithSegments;
    segment: number;
    counts: RingCounts;
    original: RingCounts;
  } | null>(null);
  let confirmDiscard = $state(false);
  let openedStored = $state("");
  let opener: HTMLElement | null = null;
  let countsError = $state("");
  let error = $state("");
  let archivedBlock = $state<Shooter | null>(null);

  const nextLane = $derived.by(() => {
    const taken = new Set(entries.filter((e) => !e.withdrawn).map((e) => e.lane));
    let n = 1;
    while (taken.has(n)) n++;
    return n;
  });
  const entered = $derived(entries.map((e) => e.shooter_id));
  const lane = $derived(laneInput ?? nextLane);
  const categories = $derived(courseCategories(course));
  const category = $derived(
    chosenCategory !== null && categories.includes(chosenCategory)
      ? chosenCategory
      : (categories[0] ?? ""),
  );
  const shooterError = $derived(submitted && shooterId === null ? "Choose a shooter." : "");
  const laneError = $derived(
    submitted && !(Number.isInteger(Number(lane)) && Number(lane) >= 1)
      ? "Lane must be a whole number from 1."
      : "",
  );
  const segmentNumbers = $derived(entries[0]?.segments.map((s) => s.segment_number) ?? []);
  const shooterName = (id: number) => {
    const s = shooters.find((x) => x.id === id) ?? imported.find((x) => x.id === id);
    return s ? `${s.first_name} ${s.last_name}` : `#${id}`;
  };
  const ordered = $derived([...entries].sort((a, b) => a.lane - b.lane || a.id - b.id));
  const commands = $derived(entryCommands(matchId, reload));
  const categoryItems = (current?: string) =>
    (current && !categories.includes(current) ? [current, ...categories] : categories).map((c) => ({
      value: c,
      label: c,
    }));

  async function run(command: Promise<string | null>): Promise<boolean> {
    error = "";
    const failed = await command;
    if (failed !== null) error = failed;
    return failed === null;
  }

  function importSaved(s: Shooter) {
    imported = [...imported.filter((x) => x.id !== s.id), s];
    chosen = s;
    shooterId = s.id;
    importQuery = null;
    pickerKey++;
  }

  async function add(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    if (shooterId === null || laneError) return;
    archivedBlock = null;
    const ok = await run(commands.add({ shooter_id: shooterId, category, lane }));
    if (!ok && error === "shooter_archived") {
      error = "";
      archivedBlock = chosen;
    }
    if (ok) {
      laneInput = null;
      shooterId = null;
      chosen = null;
      submitted = false;
      pickerKey++;
    }
  }

  async function unarchive() {
    if (!archivedBlock) return;
    const result = await settle(
      api.POST("/api/shooters/{shooterID}/unarchive", {
        params: { path: { shooterID: archivedBlock.id } },
      }),
    );
    if (result.ok) archivedBlock = null;
    else error = result.error;
  }

  async function saveEdit() {
    if (!editing) return;
    const ok = await run(
      commands.update(editing.id, { category: editing.category, lane: editing.lane }),
    );
    if (ok) editing = null;
  }

  async function saveRemarks(entry: EntryWithSegments) {
    if (!remarking) return;
    const ok = await run(
      commands.update(entry.id, {
        category: entry.category,
        lane: entry.lane,
        remarks: remarking.text,
      }),
    );
    if (ok) remarking = null;
  }

  const classificationOptions = [
    { value: 0, name: "Unclassified" },
    { value: 1, name: "Marksman" },
    { value: 2, name: "Sharpshooter" },
    { value: 3, name: "Expert" },
    { value: 4, name: "Master" },
    { value: 5, name: "High Master" },
  ];
  const classificationItems = classificationOptions.map((o) => ({
    value: String(o.value),
    label: o.name,
  }));

  async function saveCorrection() {
    if (!correcting) return;
    const { id, classification, class: competed } = correcting;
    const ok = await run(commands.classify(id, Number(classification), Number(competed)));
    if (ok) correcting = null;
  }

  async function confirmed() {
    if (!confirming) return;
    const { kind, entry } = confirming;
    confirming = null;
    await run(kind === "withdraw" ? commands.withdraw(entry.id) : commands.remove(entry.id));
  }

  const unlock = (entry: EntryWithSegments, segment: number) =>
    run(commands.unlock(entry.id, segment));
  const accept = (entry: EntryWithSegments, segment: number) =>
    run(commands.accept(entry.id, segment));
  const lock = (entry: EntryWithSegments, segment: number) => run(commands.lock(entry.id, segment));

  function openCounts(entry: EntryWithSegments, segment: number, button: HTMLElement) {
    opener = button;
    countsError = "";
    error = "";
    confirmDiscard = false;
    const stored = entry.segments.find((s) => s.segment_number === segment);
    const counts = blankCounts();
    for (const k of ringKeys) counts[k] = stored?.[k] ?? 0;
    scoring = { entry, segment, counts, original: { ...counts } };
    openedStored = storedKey(stored);
  }

  const storedKey = (s: { [k in (typeof ringKeys)[number]]: number } | undefined) =>
    ringKeys.map((k) => s?.[k] ?? 0).join(",");

  const statusLabels = {
    open: "Open",
    awaiting_acceptance: "Awaiting acceptance",
    accepted: "Accepted",
  } as const;

  async function closeCounts() {
    scoring = null;
    countsError = "";
    confirmDiscard = false;
    await tick();
    opener?.focus();
    opener = null;
  }

  function requestClose() {
    if (!scoring) return;
    const { counts, original } = scoring;
    if (ringKeys.some((k) => (Number(counts[k]) || 0) !== original[k])) {
      confirmDiscard = true;
      return;
    }
    void closeCounts();
  }

  async function saveCounts(andLock = false) {
    if (!scoring) return;
    const { entry, segment, counts } = scoring;
    countsError = (await commands.saveCounts(entry.id, segment, counts)) ?? "";
    if (countsError) return;
    await closeCounts();
    if (andLock) await lock(entry, segment);
  }

  const cell = "block p-0";
</script>

<section class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card">
  <h2 class="m-0 text-lg">Entries and progress</h2>
  {#if error}<ErrorState title={error} />{/if}
  {#if archivedBlock}
    <ErrorState title="This shooter is archived">
      {#snippet action()}
        <Button variant="secondary" onclick={unarchive}>Unarchive</Button>
      {/snippet}
    </ErrorState>
  {/if}

  {#if status === "draft"}
    <form
      class="grid items-start gap-3 rounded-lg bg-surface-alt p-3 sm:grid-cols-[minmax(0,2fr)_minmax(0,1.5fr)_6rem_auto]"
      aria-label="Add entry"
      onsubmit={add}
      novalidate
    >
      {#key pickerKey}
        <ShooterPicker
          exclude={entered}
          required
          error={shooterError}
          initial={chosen}
          onselect={(s) => {
            shooterId = s?.id ?? null;
            chosen = s;
            archivedBlock = null;
          }}
          onadd={(q) => (importQuery = q)}
        />
      {/key}
      <Field label="Category">
        <Select
          label="Entry category"
          bind:value={() => category, (v) => (chosenCategory = v)}
          items={categoryItems()}
        />
      </Field>
      <Field label="Lane" error={laneError}>
        <input
          type="number"
          min="1"
          aria-label="Entry lane"
          aria-invalid={laneError ? true : undefined}
          class={numberInput}
          bind:value={() => lane, (v) => (laneInput = v)}
          required
        />
      </Field>
      <div class="flex flex-wrap items-center gap-3 sm:pt-6">
        <Button type="submit">Add entry</Button>
        <a
          href="/admin/shooters"
          class="inline-flex min-h-11 items-center text-link outline-offset-2 focus-visible:outline-3 focus-visible:outline-focus"
          >Manage shooters</a
        >
      </div>
    </form>
    {#if importQuery !== null}
      <Wa1500Import
        query={importQuery}
        onsaved={importSaved}
        oncancel={() => (importQuery = null)}
      />
    {/if}
  {/if}

  <table class="block w-full">
    <thead class="sr-only">
      <tr>
        <th>Lane</th><th>Shooter</th><th>Category</th><th>Class</th><th>Remarks</th>
        {#each segmentNumbers as n (n)}<th title={segmentLabel(course, n)}
            >{segmentColumn(course, n)}</th
          >{/each}
        <th>Actions</th>
      </tr>
    </thead>
    <tbody class="flex flex-col gap-3">
      {#each ordered as entry (entry.id)}
        {@const name = shooterName(entry.shooter_id)}
        <tr
          class={cn(
            "flex flex-wrap items-baseline gap-x-3 gap-y-2 rounded-lg border border-border bg-surface p-3 shadow-card",
            entry.withdrawn && "border-dashed bg-surface-alt text-muted",
          )}
        >
          {#if editing?.id === entry.id}
            <td class={cn(cell, "order-1 w-24 self-center")}>
              <input
                type="number"
                min="1"
                aria-label="Move lane"
                class={numberInput}
                bind:value={editing.lane}
              />
            </td>
            <td class={cn(cell, "order-2 min-w-0 flex-1 text-lg font-semibold")}>{name}</td>
            <td class={cn(cell, "order-3 min-w-56 self-center")}>
              <Select
                label="Move category"
                bind:value={editing.category}
                items={categoryItems(entry.category)}
              />
            </td>
          {:else}
            <td class={cn(cell, "order-1 self-center font-semibold text-muted")}>
              <span aria-hidden="true">Lane </span>{entry.lane}
            </td>
            <td class={cn(cell, "order-2 min-w-0 flex-1 text-lg font-semibold")}>
              {name}
              {#if entry.withdrawn}<Badge class="ml-1">Withdrawn</Badge>{/if}
            </td>
            <td class={cn(cell, "order-3 self-center text-muted")}>{entry.category}</td>
          {/if}
          <td class={cn(cell, "order-3 flex flex-wrap items-center gap-2 text-muted")}>
            {#if correcting?.id === entry.id}
              <Select
                class="w-44"
                label="Classification at match"
                bind:value={correcting.classification}
                items={classificationItems}
              />
              <Select
                class="w-44"
                label="Class"
                bind:value={correcting.class}
                items={classificationItems}
              />
              <Button size="sm" onclick={saveCorrection}>Save class</Button>
              <Button size="sm" variant="secondary" onclick={() => (correcting = null)}
                >Cancel</Button
              >
            {:else if entry.class}
              <span>{entry.classification_at_match?.name ?? "-"} / {entry.class.name}</span>
            {/if}
            {#if correcting?.id !== entry.id && entry.club !== null && course === "150"}
              <Button
                size="sm"
                variant="secondary"
                aria-label={`Correct class for ${name}`}
                onclick={() =>
                  (correcting = {
                    id: entry.id,
                    classification: String(entry.classification_at_match?.value ?? 0),
                    class: String(entry.class?.value ?? 0),
                  })}>Correct</Button
              >
            {/if}
          </td>
          <td
            class={cn(
              cell,
              "order-3 flex min-w-0 basis-full flex-wrap items-center gap-2 text-muted",
            )}
          >
            {#if remarking?.id === entry.id}
              <Input
                class="max-w-md flex-1"
                maxlength={200}
                aria-label={`Remarks for ${name}`}
                bind:value={remarking.text}
              />
              <Button size="sm" onclick={() => saveRemarks(entry)}>Save remarks</Button>
              <Button size="sm" variant="secondary" onclick={() => (remarking = null)}
                >Cancel</Button
              >
            {:else}
              {#if entry.remarks}<span>{entry.remarks}</span>{/if}
              <Button
                size="sm"
                variant="secondary"
                aria-label={`Edit remarks for ${name}`}
                onclick={() => (remarking = { id: entry.id, text: entry.remarks ?? "" })}
                >{entry.remarks ? "Edit" : "Add remarks"}</Button
              >
            {/if}
          </td>
          <td
            class={cn(
              cell,
              "order-4 grid basis-full grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5",
            )}
          >
            {#each entry.segments as st (st.segment_number)}
              {@const locked = st.status === "accepted"}
              {@const scored = st.status === "open" && st.shots_entered >= st.expected_shots}
              <div
                class={cn(
                  "flex min-w-0 flex-col items-start gap-2 rounded-md border-l-4 p-2",
                  locked && "border-success-fg bg-success-bg",
                  st.status === "awaiting_acceptance" && "border-link bg-brand-tint",
                  scored && "border-link bg-brand-tint",
                  st.status === "open" && !scored && "border-warning-fg bg-surface-alt",
                )}
                role="group"
                aria-label={`Lane ${entry.lane} ${segmentLabel(course, st.segment_number)}`}
              >
                <span class="text-sm font-semibold text-muted"
                  >{segmentLabel(course, st.segment_number)}</span
                >
                <div class="flex flex-wrap items-center gap-2">
                  {#if locked}
                    <span
                      title={st.scorer_lane != null
                        ? `Scored by lane ${st.scorer_lane} ${st.scorer_name ?? ""}`
                        : `Locked by ${st.scorer_name ?? ""}`}
                      ><Badge tone="success">Accepted</Badge></span
                    >
                  {:else if st.status === "awaiting_acceptance"}
                    <span
                      title={st.scorer_lane != null
                        ? `Scored by lane ${st.scorer_lane} ${st.scorer_name ?? ""}`
                        : `Locked by ${st.scorer_name ?? ""}`}
                      ><Badge tone="info">Awaiting acceptance</Badge></span
                    >
                  {:else if scored}
                    <Badge tone="info">Scored, not locked</Badge>
                  {:else}
                    <Badge tone="warning">Open</Badge>
                  {/if}
                  {#if st.status !== "open" || st.shots_entered > 0}
                    <span class="text-sm text-muted"
                      ><strong class="text-base text-fg">{st.score}</strong> pts</span
                    >
                  {/if}
                </div>
                {#if st.status === "open" && st.shots_entered > 0 && st.shots_entered < st.expected_shots}
                  <span class="text-sm text-muted"
                    >{st.shots_entered} / {st.expected_shots} shots</span
                  >
                {/if}
                <div class="flex flex-wrap gap-1">
                  <Button
                    size="sm"
                    variant="secondary"
                    aria-label={`Lane ${entry.lane} segment ${st.segment_number}: Edit`}
                    onclick={(e) => openCounts(entry, st.segment_number, e.currentTarget)}
                    >Edit</Button
                  >
                  {#if st.status === "awaiting_acceptance"}
                    <Button
                      size="sm"
                      variant="secondary"
                      aria-label={`Lane ${entry.lane} segment ${st.segment_number}: Accept`}
                      onclick={() => accept(entry, st.segment_number)}>Accept</Button
                    >
                  {/if}
                  {#if st.status !== "open"}
                    <Button
                      size="sm"
                      variant="secondary"
                      aria-label={`Lane ${entry.lane} segment ${st.segment_number}: Unlock`}
                      onclick={() => unlock(entry, st.segment_number)}>Unlock</Button
                    >
                  {:else}
                    <Button
                      size="sm"
                      variant="secondary"
                      aria-label={`Lane ${entry.lane} segment ${st.segment_number}: Lock`}
                      onclick={() => lock(entry, st.segment_number)}>Lock</Button
                    >
                  {/if}
                </div>
              </div>
            {/each}
          </td>
          <td
            class={cn(
              cell,
              "order-5 flex basis-full flex-wrap items-center gap-2 border-t border-border pt-3",
            )}
          >
            {#if editing?.id === entry.id}
              <Button onclick={saveEdit}>Save</Button>
              <Button variant="secondary" onclick={() => (editing = null)}>Cancel</Button>
            {:else}
              {#if status === "draft"}
                <Button
                  variant="secondary"
                  onclick={() =>
                    (editing = { id: entry.id, category: entry.category, lane: entry.lane })}
                  >Move</Button
                >
              {/if}
              {#if !entry.withdrawn && status !== "finished"}
                <Button
                  variant="secondary"
                  onclick={() => (confirming = { kind: "withdraw", entry })}>Withdraw</Button
                >
              {/if}
              <Button
                variant="danger"
                class="ml-auto"
                onclick={() => (confirming = { kind: "remove", entry })}>Remove</Button
              >
            {/if}
          </td>
        </tr>
      {:else}
        <tr>
          <td class="block rounded-lg border border-dashed border-border p-6 text-center text-muted"
            >No entries yet. Add a shooter above to get started.</td
          >
        </tr>
      {/each}
    </tbody>
  </table>
</section>

{#if confirming}
  {@const who = shooterName(confirming.entry.shooter_id)}
  <ConfirmDialog
    title={confirming.kind === "withdraw" ? "Withdraw entry" : "Remove entry"}
    description={confirming.kind === "withdraw"
      ? `Withdraw ${who}? Results are kept. The entry stays in the results marked as withdrawn and the lane is freed.`
      : `Remove ${who} and their results? Every segment score for this entry is deleted and this cannot be undone.`}
    confirmLabel={confirming.kind === "withdraw" ? "Withdraw entry" : "Remove entry"}
    tone={confirming.kind === "withdraw" ? "primary" : "danger"}
    onconfirm={confirmed}
    oncancel={() => (confirming = null)}
  />
{/if}

{#if scoring}
  {@const expected =
    scoring.entry.segments.find((s) => s.segment_number === scoring?.segment)?.expected_shots ?? 0}
  {@const stored = entries
    .find((e) => e.id === scoring?.entry.id)
    ?.segments.find((s) => s.segment_number === scoring?.segment)}
  {@const storedChanged = storedKey(stored) !== openedStored}
  {@const total = ringKeys.reduce((n, k) => n + (Number(scoring?.counts[k]) || 0), 0)}
  <Dialog
    title={`${shooterName(scoring.entry.shooter_id)}, ${segmentLabel(course, scoring.segment)}`}
    description={`Enter all counts; they must sum to ${expected} shots.`}
    closeLabel="Close"
    autofocus="input"
    class="w-[min(36rem,calc(100vw-1.5rem))]"
    onrequestclose={requestClose}
  >
    <form
      class="flex flex-col gap-3"
      onsubmit={(e) => e.preventDefault()}
      onkeydown={(e) => {
        if (e.key === "Enter" && e.target instanceof HTMLInputElement) e.preventDefault();
      }}
    >
      {#if countsError}<ErrorState title={countsError} />{/if}
      {#if confirmDiscard}
        <div
          class="flex flex-wrap items-center gap-2 rounded-lg border border-danger bg-danger-bg p-3 text-danger-fg"
          role="alertdialog"
          aria-label="Discard changes?"
        >
          <strong class="mr-auto">Discard changes?</strong>
          <Button variant="secondary" onclick={() => (confirmDiscard = false)}>Keep editing</Button>
          <Button variant="danger" onclick={closeCounts}>Discard</Button>
        </div>
      {/if}
      <section class="rounded-lg bg-surface-alt p-3" aria-label="Stored counts">
        <h3 class="m-0 mb-2 text-base">Stored counts</h3>
        {#if storedChanged}
          <p class="m-0 mb-2 font-semibold text-danger-fg" role="status">
            Stored counts changed since you opened this
          </p>
        {/if}
        {#if stored && stored.shots_entered > 0}
          <ul class="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 p-0">
            {#each ringKeys as k (k)}
              <li>{k === "x" ? "X" : k} <strong>{stored[k]}</strong></li>
            {/each}
          </ul>
          <p class="m-0 mt-2">
            <strong>{stored.shots_entered}</strong> / {expected} shots ·
            {statusLabels[stored.status]}
          </p>
        {:else}
          <p class="m-0 text-muted">No counts entered yet</p>
        {/if}
      </section>
      <div class="grid grid-cols-[repeat(auto-fit,minmax(5rem,1fr))] gap-3">
        {#each ringKeys as k (k)}
          <label class="flex flex-col gap-1 font-bold">
            {k === "x" ? "X" : k}
            <input
              type="number"
              inputmode="numeric"
              min="0"
              class={cn(numberInput, "min-h-13 text-center text-xl")}
              bind:value={scoring.counts[k]}
            />
          </label>
        {/each}
      </div>
      <p class="m-0 flex flex-wrap items-center gap-2 text-lg">
        <strong class={cn("text-3xl", total !== expected && "text-danger-fg")}>{total}</strong>
        / {expected}
        {#if total === expected}
          <Badge tone="success">✓ Shots match</Badge>
        {:else if total > expected}
          <Badge tone="danger">⚠ {total - expected} too many</Badge>
        {:else}
          <Badge tone="warning">⚠ {expected - total} to go</Badge>
        {/if}
      </p>
      <div class="flex flex-wrap gap-2">
        <Button class="flex-1" onclick={() => saveCounts()}>Save counts</Button>
        <Button class="flex-1" onclick={() => saveCounts(true)}>Save and lock</Button>
        <Button class="flex-1" variant="secondary" onclick={requestClose}>Cancel</Button>
      </div>
    </form>
  </Dialog>
{/if}
