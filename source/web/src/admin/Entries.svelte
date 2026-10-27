<script lang="ts">
  import { entryCommands } from "../lib/entries";
  import { courseCategories, stageLabel } from "../lib/rulebook.svelte";
  import ShooterPicker from "./ShooterPicker.svelte";
  import Wa1500Import from "./Wa1500Import.svelte";
  import type { components } from "../lib/api/schema";

  type Shooter = components["schemas"]["Shooter"];
  type EntryWithStages = components["schemas"]["EntryWithStages"];
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
    entries: EntryWithStages[];
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
  let scoring = $state<{ entry: EntryWithStages; stage: number; counts: RingCounts } | null>(null);
  let error = $state("");

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
  const stageNumbers = $derived(entries[0]?.stages.map((s) => s.stage_number) ?? []);
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
    const ok = await run(commands.add({ shooter_id: shooterId, category, lane }));
    if (ok) {
      laneInput = null;
      shooterId = null;
      chosen = null;
      pickerKey++;
    }
  }

  async function saveEdit() {
    if (!editing) return;
    const ok = await run(
      commands.update(editing.id, { category: editing.category, lane: editing.lane }),
    );
    if (ok) editing = null;
  }

  async function withdraw(entry: EntryWithStages) {
    if (!confirm(`Withdraw ${shooterName(entry.shooter_id)}? Results are kept.`)) return;
    await run(commands.withdraw(entry.id));
  }

  async function remove(entry: EntryWithStages) {
    if (!confirm(`Remove ${shooterName(entry.shooter_id)} and their results?`)) return;
    await run(commands.remove(entry.id));
  }

  const unlock = (entry: EntryWithStages, stage: number) => run(commands.unlock(entry.id, stage));
  const accept = (entry: EntryWithStages, stage: number) => run(commands.accept(entry.id, stage));
  const lock = (entry: EntryWithStages, stage: number) => run(commands.lock(entry.id, stage));

  async function saveCounts(andLock = false) {
    if (!scoring) return;
    const { entry, stage, counts } = scoring;
    const ok = await run(commands.saveCounts(entry.id, stage, counts));
    if (!ok) return;
    scoring = null;
    if (andLock) await lock(entry, stage);
  }
</script>

<section class="card">
  <h2>Entries and progress</h2>
  {#if error}<p class="error" role="alert">{error}</p>{/if}

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
          {#each stageNumbers as n (n)}<th title={stageLabel(course, n)}>S{n}</th>{/each}
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
            {#each entry.stages as st (st.stage_number)}
              <td
                class="stage"
                class:state-locked={st.status === "accepted"}
                class:state-scored={st.status === "open" && st.shots_entered >= st.expected_shots}
                class:state-open={st.status === "open" && st.shots_entered < st.expected_shots}
                data-label={stageLabel(course, st.stage_number)}
              >
                <div class="stage-cell">
                  <div class="stage-status">
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
                  <span class="stage-actions">
                    <button
                      class="secondary"
                      aria-label={`Lane ${entry.lane} stage ${st.stage_number}: Edit`}
                      onclick={() =>
                        (scoring = { entry, stage: st.stage_number, counts: blankCounts() })}
                      >Edit</button
                    >
                    {#if st.status === "awaiting_acceptance"}
                      <button
                        class="secondary"
                        aria-label={`Lane ${entry.lane} stage ${st.stage_number}: Accept`}
                        onclick={() => accept(entry, st.stage_number)}>Accept</button
                      >
                    {/if}
                    {#if st.status !== "open"}
                      <button
                        class="secondary"
                        aria-label={`Lane ${entry.lane} stage ${st.stage_number}: Unlock`}
                        onclick={() => unlock(entry, st.stage_number)}>Unlock</button
                      >
                    {:else}
                      <button
                        class="secondary"
                        aria-label={`Lane ${entry.lane} stage ${st.stage_number}: Lock`}
                        onclick={() => lock(entry, st.stage_number)}>Lock</button
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
            ><td colspan={4 + stageNumbers.length}
              >No entries yet. Add a shooter above to get started.</td
            ></tr
          >
        {/each}
      </tbody>
    </table>
  </div>

  {#if scoring}
    {@const expected =
      scoring.entry.stages.find((s) => s.stage_number === scoring?.stage)?.expected_shots ?? 0}
    {@const total = ringKeys.reduce((n, k) => n + (Number(scoring?.counts[k]) || 0), 0)}
    <div class="card counts">
      <h3>{shooterName(scoring.entry.shooter_id)}, stage {scoring.stage}: overwrite ring counts</h3>
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
        <button class="primary" onclick={() => saveCounts()}>Save counts</button>
        <button class="primary" onclick={() => saveCounts(true)}>Save and lock</button>
        <button class="secondary" onclick={() => (scoring = null)}>Cancel</button>
      </div>
    </div>
  {/if}
</section>

<style>
  .add-form label {
    min-width: 0;
  }
  .entries .stage-cell {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-2);
  }
  .entries .stage-status {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }
  .entries .stage-actions,
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
  .entries .stage {
    order: 4;
    flex: 0 0 calc((100% - (var(--cols) - 1) * var(--space-2)) / var(--cols));
    box-sizing: border-box;
    padding: var(--space-2);
    background: var(--surface-alt);
    border-left: 4px solid var(--border);
    border-radius: var(--radius);
  }
  .entries .stage.state-locked {
    background: var(--success-bg);
    border-left-color: var(--success-fg);
  }
  .entries .stage.state-scored {
    background: var(--brand-tint);
    border-left-color: var(--link);
  }
  .entries .stage.state-open {
    border-left-color: var(--warning-fg);
  }
  .entries .stage.state-locked::before {
    color: var(--success-fg);
  }
  .entries .stage::before {
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
