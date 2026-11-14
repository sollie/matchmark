<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { club, clubLogoUrl, refreshClub } from "../lib/club.svelte";
  import type { components } from "../lib/api/schema";

  type Range = components["schemas"]["Range"];

  let name = $state("");
  let ranges = $state<Range[]>([]);
  let newRangeName = $state("");
  let editingRange = $state<number | null>(null);
  let editRangeName = $state("");
  let loaded = $state(false);
  let error = $state("");
  let message = $state("");
  let fileInput = $state<HTMLInputElement>();

  onMount(async () => {
    await refreshClub();
    name = club.name;
    await loadRanges();
    loaded = true;
  });

  async function loadRanges() {
    const { data } = await api.GET("/api/ranges");
    if (data) ranges = data.ranges;
  }

  async function runRangeAction(
    action: () => Promise<{ response: Response } & object>,
    ok: string,
  ) {
    error = "";
    message = "";
    const result = await action();
    if (!result.response.ok) {
      error = failure(result as Parameters<typeof failure>[0]);
      return false;
    }
    message = ok;
    await Promise.all([loadRanges(), refreshClub()]);
    return true;
  }

  async function addRange(e: SubmitEvent) {
    e.preventDefault();
    if (
      await runRangeAction(
        () => api.POST("/api/ranges", { body: { name: newRangeName } }),
        "Range added.",
      )
    ) {
      newRangeName = "";
    }
  }

  function startRename(range: Range) {
    editingRange = range.id;
    editRangeName = range.name;
  }

  async function renameRange(e: SubmitEvent, range: Range) {
    e.preventDefault();
    const params = { path: { rangeID: range.id } };
    if (
      await runRangeAction(
        () => api.PUT("/api/ranges/{rangeID}", { params, body: { name: editRangeName } }),
        "Range renamed.",
      )
    ) {
      editingRange = null;
    }
  }

  function archiveRange(range: Range, archive: boolean) {
    const params = { path: { rangeID: range.id } };
    return runRangeAction(
      () =>
        archive
          ? api.POST("/api/ranges/{rangeID}/archive", { params })
          : api.POST("/api/ranges/{rangeID}/unarchive", { params }),
      archive ? "Range archived." : "Range restored.",
    );
  }

  function deleteRange(range: Range) {
    if (!confirm(`Delete range "${range.name}"?`)) return;
    return runRangeAction(
      () => api.DELETE("/api/ranges/{rangeID}", { params: { path: { rangeID: range.id } } }),
      "Range deleted.",
    );
  }

  async function chooseDefaultRange(e: Event) {
    const select = e.currentTarget as HTMLSelectElement;
    const value = select.value === "" ? null : Number(select.value);
    const ok = await runRangeAction(
      () => api.PUT("/api/club", { body: { default_range_id: value } }),
      "Default range saved.",
    );
    if (!ok) select.value = club.defaultRangeId === null ? "" : String(club.defaultRangeId);
  }

  async function chooseUnclassifiedClass(e: Event) {
    const select = e.currentTarget as HTMLSelectElement;
    error = "";
    message = "";
    const result = await api.PUT("/api/club", {
      body: { unclassified_class: select.value as "Master" | "Unclassified" },
    });
    if (!result.response.ok) {
      select.value = club.unclassifiedClass;
      error = failure(result);
      return;
    }
    message = "Unclassified class saved.";
    await refreshClub();
  }

  async function saveName(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    message = "";
    const result = await api.PUT("/api/club", { body: { name } });
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    message = "Club name saved.";
    await refreshClub();
    name = club.name;
  }

  async function uploadLogo(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    message = "";
    const file = fileInput?.files?.[0];
    if (!file) {
      error = "Choose a PNG or JPEG file first.";
      return;
    }
    const response = await fetch("/api/club/logo", {
      method: "PUT",
      credentials: "same-origin",
      headers: { "Content-Type": file.type },
      body: file,
    });
    if (!response.ok) {
      error =
        response.status === 422
          ? "The logo must be a PNG or JPEG of at most 512 KB and 2048 by 2048 pixels."
          : `Request failed (${response.status})`;
      return;
    }
    message = "Logo saved.";
    if (fileInput) fileInput.value = "";
    await refreshClub();
  }

  async function removeLogo() {
    error = "";
    message = "";
    const result = await api.DELETE("/api/club/logo");
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    message = "Logo removed.";
    await refreshClub();
  }
</script>

<h1>Club settings</h1>

{#if loaded}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if message}<p role="status">{message}</p>{/if}

  <section aria-labelledby="club-name-heading">
    <h2 id="club-name-heading">Club name</h2>
    <form onsubmit={saveName}>
      <label>Name <input bind:value={name} maxlength="100" /></label>
      <button type="submit">Save name</button>
    </form>
  </section>

  <section aria-labelledby="club-class-heading">
    <h2 id="club-class-heading">Unclassified shooters</h2>
    <label>
      Unclassified shooters compete as
      <select value={club.unclassifiedClass} onchange={chooseUnclassifiedClass}>
        <option value="Master">Master</option>
        <option value="Unclassified">Unclassified</option>
      </select>
    </label>
    <p class="muted">Applied when a match goes active (rule 20.2).</p>
  </section>

  <section aria-labelledby="club-ranges-heading">
    <h2 id="club-ranges-heading">Ranges</h2>
    <label>
      Default range
      <select
        value={club.defaultRangeId === null ? "" : String(club.defaultRangeId)}
        onchange={chooseDefaultRange}
      >
        <option value="">None</option>
        {#each ranges.filter((r) => !r.archived) as range (range.id)}
          <option value={String(range.id)}>{range.name}</option>
        {/each}
      </select>
    </label>
    <p class="muted">
      Pre-filled on new competitions. A club with one active range selects it automatically.
    </p>
    {#if ranges.length === 0}
      <p class="muted">No ranges yet.</p>
    {:else}
      <ul>
        {#each ranges as range (range.id)}
          <li aria-label={`Range ${range.name}`}>
            {#if editingRange === range.id}
              <form class="row" onsubmit={(e) => renameRange(e, range)}>
                <label
                  >Range name <input bind:value={editRangeName} maxlength="100" required /></label
                >
                <button type="submit">Save</button>
                <button type="button" onclick={() => (editingRange = null)}>Cancel</button>
              </form>
            {:else}
              <span class="row">
                {range.name}
                {#if range.archived}<span class="muted">(archived)</span>{/if}
                {#if range.id === club.defaultRangeId}<span class="muted">(default)</span>{/if}
                <button
                  type="button"
                  onclick={() => startRename(range)}
                  aria-label={`Rename range ${range.name}`}>Rename</button
                >
                {#if range.archived}
                  <button
                    type="button"
                    onclick={() => archiveRange(range, false)}
                    aria-label={`Restore range ${range.name}`}>Restore</button
                  >
                {:else if range.id !== club.defaultRangeId}
                  <button
                    type="button"
                    onclick={() => archiveRange(range, true)}
                    aria-label={`Archive range ${range.name}`}>Archive</button
                  >
                {/if}
                {#if range.id !== club.defaultRangeId}
                  <button
                    type="button"
                    onclick={() => deleteRange(range)}
                    aria-label={`Delete range ${range.name}`}>Delete</button
                  >
                {/if}
              </span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
    <form onsubmit={addRange}>
      <label>Range to add <input bind:value={newRangeName} maxlength="100" required /></label>
      <button type="submit">Add range</button>
    </form>
  </section>

  <section aria-labelledby="club-logo-heading">
    <h2 id="club-logo-heading">Club logo</h2>
    {#if club.hasLogo}
      <p>
        <img class="club-logo-preview" src={clubLogoUrl(club.version)} alt="Current club logo" />
      </p>
      <button type="button" onclick={removeLogo}>Remove logo</button>
    {:else}
      <p class="muted">No logo uploaded.</p>
    {/if}
    <form onsubmit={uploadLogo}>
      <label
        >Logo file <input type="file" accept="image/png,image/jpeg" bind:this={fileInput} /></label
      >
      <button type="submit">Upload logo</button>
    </form>
    <p class="muted">PNG or JPEG, at most 512 KB and 2048 by 2048 pixels.</p>
  </section>
{/if}
