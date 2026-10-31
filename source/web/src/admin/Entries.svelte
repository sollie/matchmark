<script lang="ts">
  import { tick } from "svelte";
  import { api } from "../lib/api/client";
  import { settle } from "../lib/api/util";
  import { entryCommands } from "../lib/entries";
  import { courseCategories, segmentColumn, segmentLabel } from "../lib/rulebook.svelte";
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

  let shooterId = $state<number | null>(null);
  let pickerKey = $state(0);
  let importQuery = $state<string | null>(null);
  let imported = $state<Shooter[]>([]);
  let chosen = $state<Shooter | null>(null);
  let chosenCategory = $state<string | null>(null);
  let laneInput = $state<number | null>(null);
  let editing = $state<{ id: number; category: string; lane: number } | null>(null);
  let scoring = $state<{ entry: EntryWithSegments; segment: number; counts: RingCounts } | null>(
    null,
  );
  let countsDialog = $state<HTMLDialogElement>();
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
  const categories = $derived([...courseCategories(course)].sort((a, b) => a.localeCompare(b)));
  const category = $derived(
    chosenCategory !== null && categories.includes(chosenCategory)
      ? chosenCategory
      : (categories[0] ?? ""),
  );
  const segmentNumbers = $derived(entries[0]?.segments.map((s) => s.segment_number) ?? []);
  const shooterName = (id: number) => {
    const s = shooters.find((x) => x.id === id) ?? imported.find((x) => x.id === id);
    return s ? `${s.first_name} ${s.last_name}` : `#${id}`;
  };
  const ordered = $derived([...entries].sort((a, b) => a.lane - b.lane || a.id - b.id));
  const commands = $derived(entryCommands(matchId, reload));

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
    if (shooterId === null) return;
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

  async function withdraw(entry: EntryWithSegments) {
    if (!confirm(`Withdraw ${shooterName(entry.shooter_id)}? Results are kept.`)) return;
    await run(commands.withdraw(entry.id));
  }

  async function remove(entry: EntryWithSegments) {
    if (!confirm(`Remove ${shooterName(entry.shooter_id)} and their results?`)) return;
    await run(commands.remove(entry.id));
  }

  const unlock = (entry: EntryWithSegments, segment: number) =>
    run(commands.unlock(entry.id, segment));
  const accept = (entry: EntryWithSegments, segment: number) =>
    run(commands.accept(entry.id, segment));
  const lock = (entry: EntryWithSegments, segment: number) => run(commands.lock(entry.id, segment));

  async function openCounts(entry: EntryWithSegments, segment: number, button: HTMLElement) {
    opener = button;
    countsError = "";
    const stored = entry.segments.find((s) => s.segment_number === segment);
    const counts = blankCounts();
    for (const k of ringKeys) counts[k] = stored?.[k] ?? 0;
    scoring = { entry, segment, counts };
    openedStored = storedKey(stored);
    await tick();
    countsDialog?.showModal();
    countsDialog?.querySelector("input")?.focus();
  }

  const storedKey = (s: { [k in (typeof ringKeys)[number]]: number } | undefined) =>
    ringKeys.map((k) => s?.[k] ?? 0).join(",");

  const statusLabels = {
    open: "Open",
    awaiting_acceptance: "Awaiting acceptance",
    accepted: "Accepted",
  } as const;

  function closeCounts() {
    countsDialog?.close();
  }

  function counted() {
    scoring = null;
    countsError = "";
    opener?.focus();
    opener = null;
  }

  async function saveCounts(andLock = false) {
    if (!scoring) return;
    const { entry, segment, counts } = scoring;
    countsError = (await commands.saveCounts(entry.id, segment, counts)) ?? "";
    if (countsError) return;
    closeCounts();
    if (andLock) await lock(entry, segment);
  }
</script>

<section class="card">
  <h2>Entries and progress</h2>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if archivedBlock}
    <p class="error row" role="alert">
      This shooter is archived
      <button type="button" class="secondary" onclick={unarchive}>Unarchive</button>
    </p>
  {/if}

  {#if status === "draft"}
    <form class="row add-form" aria-label="Add entry" onsubmit={add}>
      {#key pickerKey}
        <ShooterPicker
          exclude={entered}
          required
          initial={chosen}
          onselect={(s) => {
            shooterId = s?.id ?? null;
            chosen = s;
            archivedBlock = null;
          }}
          onadd={(q) => (importQuery = q)}
        />
      {/key}
      <label>
        Category
        <select
          aria-label="Entry category"
          bind:value={() => category, (v) => (chosenCategory = v)}
        >
          {#each categories as c (c)}<option value={c}>{c}</option>{/each}
        </select>
      </label>
      <label
        >Lane <input
          type="number"
          min="1"
          aria-label="Entry lane"
          bind:value={() => lane, (v) => (laneInput = v)}
          size="3"
          required
        /></label
      >
      <button type="submit" class="primary">Add entry</button>
      <a href="/admin/shooters">Manage shooters</a>
    </form>
    {#if importQuery !== null}
      <Wa1500Import
        query={importQuery}
        onsaved={importSaved}
        oncancel={() => (importQuery = null)}
      />
    {/if}
  {/if}

  <div class="scroll">
    <table class="entries">
      <thead>
        <tr>
          <th>Lane</th><th>Shooter</th><th>Category</th>
          {#each segmentNumbers as n (n)}<th title={segmentLabel(course, n)}
              >{segmentColumn(course, n)}</th
            >{/each}
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each ordered as entry (entry.id)}
          <tr class:muted={entry.withdrawn}>
            {#if editing?.id === entry.id}
              <td class="lane"
                ><input
                  type="number"
                  min="1"
                  aria-label="Move lane"
                  bind:value={editing.lane}
                  size="3"
                /></td
              >
              <td class="shooter">{shooterName(entry.shooter_id)}</td>
              <td class="category">
                <select aria-label="Move category" bind:value={editing.category}>
                  {#each categories.includes(entry.category) ? categories : [entry.category, ...categories] as c (c)}<option
                      value={c}>{c}</option
                    >{/each}
                </select>
              </td>
            {:else}
              <td class="lane">{entry.lane}</td>
              <td class="shooter">
                {shooterName(entry.shooter_id)}
                {#if entry.withdrawn}<span class="badge neutral">Withdrawn</span>{/if}
              </td>
              <td class="category">{entry.category}</td>
            {/if}
            {#each entry.segments as st (st.segment_number)}
              <td
                class="segment"
                class:state-locked={st.status === "accepted"}
                class:state-scored={st.status === "open" && st.shots_entered >= st.expected_shots}
                class:state-open={st.status === "open" && st.shots_entered < st.expected_shots}
                data-label={segmentLabel(course, st.segment_number)}
              >
                <div class="segment-cell">
                  <div class="segment-status">
                    {#if st.status === "accepted"}
                      <span
                        class="badge success"
                        title={st.scorer_lane != null
                          ? `Scored by lane ${st.scorer_lane} ${st.scorer_name ?? ""}`
                          : `Locked by ${st.scorer_name ?? ""}`}>Accepted</span
                      >
                    {:else if st.status === "awaiting_acceptance"}
                      <span
                        class="badge info"
                        title={st.scorer_lane != null
                          ? `Scored by lane ${st.scorer_lane} ${st.scorer_name ?? ""}`
                          : `Locked by ${st.scorer_name ?? ""}`}>Awaiting acceptance</span
                      >
                    {:else if st.shots_entered >= st.expected_shots}
                      <span class="badge info">Scored, not locked</span>
                    {:else}
                      <span class="badge warning">Open</span>
                    {/if}
                    {#if st.status !== "open" || st.shots_entered > 0}
                      <span class="score"><strong>{st.score}</strong> pts</span>
                    {/if}
                  </div>
                  {#if st.status === "open" && st.shots_entered > 0 && st.shots_entered < st.expected_shots}
                    <span class="muted">{st.shots_entered} / {st.expected_shots} shots</span>
                  {/if}
                  <span class="segment-actions">
                    <button
                      class="secondary"
                      aria-label={`Lane ${entry.lane} segment ${st.segment_number}: Edit`}
                      onclick={(e) => openCounts(entry, st.segment_number, e.currentTarget)}
                      >Edit</button
                    >
                    {#if st.status === "awaiting_acceptance"}
                      <button
                        class="secondary"
                        aria-label={`Lane ${entry.lane} segment ${st.segment_number}: Accept`}
                        onclick={() => accept(entry, st.segment_number)}>Accept</button
                      >
                    {/if}
                    {#if st.status !== "open"}
                      <button
                        class="secondary"
                        aria-label={`Lane ${entry.lane} segment ${st.segment_number}: Unlock`}
                        onclick={() => unlock(entry, st.segment_number)}>Unlock</button
                      >
                    {:else}
                      <button
                        class="secondary"
                        aria-label={`Lane ${entry.lane} segment ${st.segment_number}: Lock`}
                        onclick={() => lock(entry, st.segment_number)}>Lock</button
                      >
                    {/if}
                  </span>
                </div>
              </td>
            {/each}
            <td class="entry-actions">
              <div class="entry-buttons">
                {#if editing?.id === entry.id}
                  <button class="primary" onclick={saveEdit}>Save</button>
                  <button class="secondary" onclick={() => (editing = null)}>Cancel</button>
                {:else}
                  {#if status === "draft"}
                    <button
                      class="secondary"
                      onclick={() =>
                        (editing = { id: entry.id, category: entry.category, lane: entry.lane })}
                    >
                      Move
                    </button>
                  {/if}
                  {#if !entry.withdrawn && status !== "finished"}<button
                      class="secondary"
                      onclick={() => withdraw(entry)}>Withdraw</button
                    >{/if}
                  <button class="danger remove" onclick={() => remove(entry)}>Remove</button>
                {/if}
              </div>
            </td>
          </tr>
        {:else}
          <tr class="empty"
            ><td colspan={4 + segmentNumbers.length}
              >No entries yet. Add a shooter above to get started.</td
            ></tr
          >
        {/each}
      </tbody>
    </table>
  </div>

  <dialog bind:this={countsDialog} class="counts" aria-labelledby="counts-title" onclose={counted}>
    {#if scoring}
      {@const expected =
        scoring.entry.segments.find((s) => s.segment_number === scoring?.segment)?.expected_shots ??
        0}
      {@const stored = entries
        .find((e) => e.id === scoring?.entry.id)
        ?.segments.find((s) => s.segment_number === scoring?.segment)}
      {@const storedChanged = storedKey(stored) !== openedStored}
      {@const total = ringKeys.reduce((n, k) => n + (Number(scoring?.counts[k]) || 0), 0)}
      <form
        onsubmit={(e) => e.preventDefault()}
        onkeydown={(e) => {
          if (e.key === "Enter" && e.target instanceof HTMLInputElement) e.preventDefault();
        }}
      >
        <div class="counts-head">
          <h3 id="counts-title">
            {shooterName(scoring.entry.shooter_id)}, {segmentLabel(course, scoring.segment)}
          </h3>
          <button type="button" class="secondary" aria-label="Close" onclick={closeCounts}>✕</button
          >
        </div>
        {#if countsError}<p class="error" role="alert">{countsError}</p>{/if}
        <section class="stored" aria-label="Stored counts">
          <h4>Stored counts</h4>
          {#if storedChanged}
            <p class="error" role="status">Stored counts changed since you opened this</p>
          {/if}
          {#if stored && stored.shots_entered > 0}
            <ul class="stored-grid">
              {#each ringKeys as k (k)}
                <li>{k === "x" ? "X" : k} <strong>{stored[k]}</strong></li>
              {/each}
            </ul>
            <p>
              <strong>{stored.shots_entered}</strong> / {expected} shots ·
              {statusLabels[stored.status]}
            </p>
          {:else}
            <p class="muted">No counts entered yet</p>
          {/if}
        </section>
        <p class="muted">Enter all counts; they must sum to {expected} shots.</p>
        <div class="count-grid">
          {#each ringKeys as k (k)}
            <label>
              {k === "x" ? "X" : k}
              <input type="number" inputmode="numeric" min="0" bind:value={scoring.counts[k]} />
            </label>
          {/each}
        </div>
        <p class="total" class:mismatch={total !== expected}>
          <strong>{total}</strong> / {expected}
          {#if total === expected}
            <span class="badge success">✓ Shots match</span>
          {:else if total > expected}
            <span class="badge danger">⚠ {total - expected} too many</span>
          {:else}
            <span class="badge warning">⚠ {expected - total} to go</span>
          {/if}
        </p>
        <div class="count-actions">
          <button type="button" class="primary" onclick={() => saveCounts()}>Save counts</button>
          <button type="button" class="primary" onclick={() => saveCounts(true)}
            >Save and lock</button
          >
          <button type="button" class="secondary" onclick={closeCounts}>Cancel</button>
        </div>
      </form>
    {/if}
  </dialog>
</section>

<style>
  .add-form label {
    min-width: 0;
  }
  .entries .segment-cell {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-2);
  }
  .entries .segment-status {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }
  .entries .segment-actions,
  .entries .entry-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }
  .entries .entry-buttons .remove {
    margin-left: var(--space-3);
  }
  .entries .shooter {
    font-weight: 600;
  }
  .entries .score {
    font-size: var(--text-sm);
    color: var(--text-muted);
  }
  .entries .score strong {
    color: var(--text);
    font-size: var(--text-base);
  }
  .entries .shooter .badge {
    margin-left: var(--space-1);
  }
  .entries .empty td {
    padding: var(--space-5);
    text-align: center;
    color: var(--text-muted);
  }
  dialog.counts {
    width: min(36rem, calc(100vw - 2 * var(--space-3)));
    max-height: calc(100vh - 2 * var(--space-3));
    overflow: auto;
    padding: var(--space-4);
    border: 1px solid var(--border, currentColor);
    border-radius: var(--radius, 0.5rem);
    background: var(--surface, Canvas);
    color: var(--text);
  }
  .counts-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-3);
  }
  .counts-head h3 {
    margin: 0;
  }
  .stored h4 {
    margin: var(--space-3) 0 var(--space-2);
  }
  .stored-grid {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-3);
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .count-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(5rem, 1fr));
    gap: var(--space-3);
    margin: var(--space-3) 0;
  }
  .count-grid label {
    font-size: var(--text-base);
    font-weight: 700;
  }
  .count-grid input {
    width: 100%;
    min-height: 3.25rem;
    font-size: 1.5rem;
    text-align: center;
  }
  .total {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
    margin: 0 0 var(--space-3);
    font-size: var(--text-lg);
  }
  .total strong {
    font-size: 2rem;
  }
  .total.mismatch strong {
    color: var(--danger-fg);
  }
  .count-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }
  .count-actions button {
    flex: 1 1 8rem;
    min-height: var(--touch);
  }
  tr.muted td {
    color: var(--text-muted);
  }

  .entries,
  .entries tbody,
  .entries tr,
  .entries td {
    display: block;
  }
  .entries thead {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }
  .entries tbody tr {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin: var(--space-3) 0;
    padding: var(--space-3);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow);
  }
  .entries tbody tr:hover > td {
    background: transparent;
  }
  .entries td {
    padding: 0;
    border: 0;
  }
  .entries tbody tr {
    --cols: 2;
    align-items: baseline;
  }
  .entries tbody tr::after {
    content: "";
    order: 3;
    flex: 0 0 100%;
    height: 0;
  }
  .entries .lane {
    order: 1;
  }
  .entries .shooter {
    order: 2;
    flex: 1 1 0;
    min-width: 0;
    font-size: var(--text-lg);
  }
  .entries .category {
    order: 3;
    color: var(--text-muted);
  }
  .entries .lane::before {
    content: "Lane ";
    color: var(--text-muted);
    font-weight: 600;
  }
  .entries .lane,
  .entries .category {
    align-self: center;
  }
  .entries .segment {
    order: 4;
    flex: 0 0 calc((100% - (var(--cols) - 1) * var(--space-2)) / var(--cols));
    box-sizing: border-box;
    padding: var(--space-2);
    background: var(--surface-alt);
    border-left: 4px solid var(--border);
    border-radius: var(--radius);
  }
  .entries .segment.state-locked {
    background: var(--success-bg);
    border-left-color: var(--success-fg);
  }
  .entries .segment.state-scored {
    background: var(--brand-tint);
    border-left-color: var(--link);
  }
  .entries .segment.state-open {
    border-left-color: var(--warning-fg);
  }
  .entries .segment.state-locked::before {
    color: var(--success-fg);
  }
  .entries .segment::before {
    content: attr(data-label);
    display: block;
    margin-bottom: var(--space-1);
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--text-muted);
  }
  .entries .entry-actions {
    order: 5;
    flex: 1 1 100%;
    padding-top: var(--space-2);
    border-top: 1px solid var(--border);
  }
  .entries .entry-buttons {
    align-items: center;
  }
  .entries .entry-buttons .remove {
    margin-left: auto;
  }
  .entries button {
    min-height: var(--touch);
    min-width: var(--touch);
  }
  .entries input,
  .entries select {
    min-height: var(--touch);
  }
  .entries .empty {
    display: block;
    padding: 0;
    border: 1px dashed var(--border);
    border-radius: var(--radius-lg);
  }
  .entries .empty td {
    padding: var(--space-5);
  }
  .scroll {
    overflow: visible;
    border: 0;
  }
  @media (min-width: 30rem) {
    .entries tbody tr {
      --cols: 3;
    }
  }
  @media (min-width: 48rem) {
    .entries tbody tr {
      --cols: 5;
    }
  }

  @media (max-width: 47.99rem) {
    .add-form {
      flex-direction: column;
      align-items: stretch;
    }
    .add-form :is(input, select, button) {
      width: 100%;
      min-height: var(--touch);
    }
    .add-form a {
      display: inline-flex;
      align-items: center;
      min-height: var(--touch);
    }
  }
</style>
