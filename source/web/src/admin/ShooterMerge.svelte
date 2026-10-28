<script lang="ts">
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import ShooterPicker from "./ShooterPicker.svelte";

  type Shooter = components["schemas"]["Shooter"];
  type Preview = components["schemas"]["ShooterMergePreview"];

  let {
    source,
    onmerged,
    oncancel,
  }: { source: Shooter; onmerged: (target: Shooter) => void; oncancel: () => void } = $props();

  let target = $state<Shooter | null>(null);
  let preview = $state<Preview | null>(null);
  let confirmed = $state(false);
  let busy = $state(false);
  let error = $state("");

  const name = (s: Shooter) => `${s.first_name} ${s.last_name}`;

  async function pick(s: Shooter | null) {
    target = s;
    preview = null;
    confirmed = false;
    error = "";
    if (!s) return;
    const result = await api.GET("/api/shooters/{shooterID}/merge-preview", {
      params: { path: { shooterID: source.id }, query: { target_id: s.id } },
    });
    if (target?.id !== s.id) return;
    if (result.data) preview = result.data;
    else error = failure(result);
  }

  async function merge() {
    if (!target || !preview) return;
    busy = true;
    error = "";
    const result = await api.POST("/api/shooters/{shooterID}/merge", {
      params: { path: { shooterID: source.id } },
      body: { target_id: target.id },
    });
    busy = false;
    if (result.response.ok) {
      onmerged(target);
      return;
    }
    if (result.response.status === 409 && result.error && "conflicts" in result.error) {
      preview = result.error;
      confirmed = false;
      return;
    }
    error = failure(result);
  }
</script>

<div class="dialog card" role="dialog" aria-modal="true" aria-labelledby="merge-title">
  <h3 id="merge-title">Merge {name(source)}</h3>
  <p>
    All entries of {name(source)} move to the shooter you pick, and {name(source)} is deleted. The target
    keeps its own details.
  </p>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <ShooterPicker label="Merge into" exclude={[source.id]} onselect={pick} />

  {#if preview && target}
    <h4>Entries to move</h4>
    {#if preview.entries.length === 0}
      <p class="muted">No entries.</p>
    {:else}
      <ul>
        {#each preview.entries as e (e.entry_id)}
          <li>{e.match_name}, {e.match_date}, {e.match_status}</li>
        {/each}
      </ul>
    {/if}

    {#if preview.waid_conflict}
      <p class="error" role="alert">
        The shooters have different WAIDs ({source.waid} and {target.waid}), so they are different
        people.
      </p>
    {:else}
      <p>
        {preview.resulting_waid
          ? `WAID after merge: ${preview.resulting_waid}`
          : "Neither shooter has a WAID."}
      </p>
    {/if}

    {#if preview.conflicts.length > 0}
      <div role="alert">
        <p class="error">Both shooters are entered in these matches. Withdraw or fix one first:</p>
        <ul>
          {#each preview.conflicts as c (c.match_id)}
            <li><a href={`/admin/matches/${c.match_id}`}>{c.match_name}</a>, {c.match_date}</li>
          {/each}
        </ul>
      </div>
    {/if}

    {#if preview.can_merge}
      <label>
        <input type="checkbox" bind:checked={confirmed} />
        I understand this cannot be undone
      </label>
    {/if}
  {/if}

  <div class="row">
    <button
      type="button"
      class="danger"
      disabled={busy || !preview?.can_merge || !confirmed}
      onclick={merge}>Merge shooters</button
    >
    <button type="button" class="secondary" onclick={oncancel}>Cancel</button>
  </div>
</div>

<style>
  .dialog {
    border: 1px solid var(--border, #ccc);
    background: var(--surface, #fff);
  }
</style>
