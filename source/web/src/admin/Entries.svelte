<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { categories } from "../lib/domain";
  import type { components } from "../lib/api/schema";

  type Shooter = components["schemas"]["Shooter"];
  type EntryWithStages = components["schemas"]["EntryWithStages"];
  type RingCounts = components["schemas"]["RingCounts"];

  let {
    competitionId,
    entries,
    shooters,
    reload,
  }: {
    competitionId: number;
    entries: EntryWithStages[];
    shooters: Shooter[];
    reload: () => Promise<void>;
  } = $props();

  const ringKeys = ["x", "10", "9", "8", "7", "0", "miss"] as const;
  const blankCounts = (): RingCounts => ({ x: 0, "10": 0, "9": 0, "8": 0, "7": 0, "0": 0, miss: 0 });

  let shooterId = $state<number | null>(null);
  let category = $state(categories[0]);
  let lane = $state(1);
  let editing = $state<{ id: number; category: string; lane: number } | null>(null);
  let scoring = $state<{ entry: EntryWithStages; stage: number; counts: RingCounts } | null>(null);
  let error = $state("");

  const stageNumbers = $derived(entries[0]?.stages.map((s) => s.stage_number) ?? []);
  const shooterName = (id: number) => {
    const s = shooters.find((x) => x.id === id);
    return s ? `${s.last_name}, ${s.first_name}` : `#${id}`;
  };
  const ordered = $derived([...entries].sort((a, b) => a.lane - b.lane || a.id - b.id));
  const path = (e: EntryWithStages) => ({ competitionID: competitionId, entryID: e.id });

  async function run(call: Promise<{ error?: unknown; response: Response }>): Promise<boolean> {
    error = "";
    const result = await call;
    if (!result.response.ok) {
      error = failure(result);
      return false;
    }
    await reload();
    return true;
  }

  async function add(e: SubmitEvent) {
    e.preventDefault();
    if (shooterId === null) return;
    const ok = await run(
      api.POST("/api/competitions/{competitionID}/entries", {
        params: { path: { competitionID: competitionId } },
        body: { shooter_id: shooterId, category, lane },
      }),
    );
    if (ok) lane = lane + 1;
  }

  async function saveEdit() {
    if (!editing) return;
    const ok = await run(
      api.PUT("/api/competitions/{competitionID}/entries/{entryID}", {
        params: { path: { competitionID: competitionId, entryID: editing.id } },
        body: { category: editing.category, lane: editing.lane },
      }),
    );
    if (ok) editing = null;
  }

  async function withdraw(entry: EntryWithStages) {
    if (!confirm(`Withdraw ${shooterName(entry.shooter_id)}? Results are kept.`)) return;
    await run(api.POST("/api/competitions/{competitionID}/entries/{entryID}/withdraw", { params: { path: path(entry) } }));
  }

  async function remove(entry: EntryWithStages) {
    if (!confirm(`Remove ${shooterName(entry.shooter_id)} and their results?`)) return;
    await run(api.DELETE("/api/competitions/{competitionID}/entries/{entryID}", { params: { path: path(entry) } }));
  }

  async function unlock(entry: EntryWithStages, stage: number) {
    await run(
      api.POST("/api/competitions/{competitionID}/entries/{entryID}/stages/{stage}/unlock", {
        params: { path: { ...path(entry), stage } },
      }),
    );
  }

  async function saveCounts() {
    if (!scoring) return;
    const ok = await run(
      api.PUT("/api/competitions/{competitionID}/entries/{entryID}/stages/{stage}", {
        params: { path: { ...path(scoring.entry), stage: scoring.stage } },
        body: scoring.counts,
      }),
    );
    if (ok) scoring = null;
  }
</script>

<section class="card">
  <h2>Entries and progress</h2>
  {#if error}<p class="error">{error}</p>{/if}

  <form class="row" onsubmit={add}>
    <label>
      Shooter
      <select bind:value={shooterId} required>
        <option value={null} disabled>Select shooter</option>
        {#each shooters as s (s.id)}<option value={s.id}>{s.last_name}, {s.first_name}</option>{/each}
      </select>
    </label>
    <label>
      Category
      <select bind:value={category}>
        {#each categories as c (c)}<option value={c}>{c}</option>{/each}
      </select>
    </label>
    <label>Lane <input type="number" min="1" bind:value={lane} size="3" required /></label>
    <button type="submit">Add entry</button>
    <a href="/admin/shooters">Manage shooters</a>
  </form>

  <table>
    <thead>
      <tr>
        <th>Lane</th><th>Shooter</th><th>Category</th>
        {#each stageNumbers as n (n)}<th>S{n}</th>{/each}
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each ordered as entry (entry.id)}
        <tr class:muted={entry.withdrawn}>
          {#if editing?.id === entry.id}
            <td><input type="number" min="1" bind:value={editing.lane} size="3" /></td>
            <td>{shooterName(entry.shooter_id)}</td>
            <td>
              <select bind:value={editing.category}>
                {#each categories as c (c)}<option value={c}>{c}</option>{/each}
              </select>
            </td>
          {:else}
            <td>{entry.lane}</td>
            <td>{shooterName(entry.shooter_id)}{entry.withdrawn ? " (withdrawn)" : ""}</td>
            <td>{entry.category}</td>
          {/if}
          {#each entry.stages as st (st.stage_number)}
            <td class={st.locked ? "locked" : "open"}>
              {#if st.locked}
                <span title={`Scored by lane ${st.scorer_lane ?? "?"} ${st.scorer_name ?? ""}`}>Locked</span>
                <button onclick={() => unlock(entry, st.stage_number)}>Unlock</button>
              {:else}
                Open
              {/if}
              <button onclick={() => (scoring = { entry, stage: st.stage_number, counts: blankCounts() })}>Edit</button>
            </td>
          {/each}
          <td>
            {#if editing?.id === entry.id}
              <button onclick={saveEdit}>Save</button>
              <button onclick={() => (editing = null)}>Cancel</button>
            {:else}
              <button onclick={() => (editing = { id: entry.id, category: entry.category, lane: entry.lane })}>
                Move
              </button>
              {#if !entry.withdrawn}<button onclick={() => withdraw(entry)}>Withdraw</button>{/if}
              <button onclick={() => remove(entry)}>Remove</button>
            {/if}
          </td>
        </tr>
      {:else}
        <tr><td colspan={4 + stageNumbers.length} class="muted">No entries yet.</td></tr>
      {/each}
    </tbody>
  </table>

  {#if scoring}
    <div class="card">
      <h3>{shooterName(scoring.entry.shooter_id)}, stage {scoring.stage}: overwrite ring counts</h3>
      <p class="muted">
        Enter all counts; they must sum to {scoring.entry.stages.find((s) => s.stage_number === scoring?.stage)
          ?.expected_shots} shots.
      </p>
      <div class="row">
        {#each ringKeys as k (k)}
          <label>{k === "x" ? "X" : k} <input type="number" min="0" bind:value={scoring.counts[k]} size="3" /></label>
        {/each}
        <button onclick={saveCounts}>Save counts</button>
        <button onclick={() => (scoring = null)}>Cancel</button>
      </div>
    </div>
  {/if}
</section>
