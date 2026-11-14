<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { club, clubLogoUrl, refreshClub } from "../lib/club.svelte";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import ConfirmDialog from "../lib/ui/ConfirmDialog.svelte";
  import EmptyState from "../lib/ui/EmptyState.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Select from "../lib/ui/Select.svelte";
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

  let pendingDelete = $state<Range | null>(null);

  function deleteRange(range: Range) {
    pendingDelete = null;
    return runRangeAction(
      () => api.DELETE("/api/ranges/{rangeID}", { params: { path: { rangeID: range.id } } }),
      "Range deleted.",
    );
  }

  const defaultRangeValue = $derived(
    club.defaultRangeId === null ? "none" : String(club.defaultRangeId),
  );
  const defaultRangeItems = $derived([
    { value: "none", label: "None" },
    ...ranges.filter((r) => !r.archived).map((r) => ({ value: String(r.id), label: r.name })),
  ]);
  const unclassifiedItems = [
    { value: "Master", label: "Master" },
    { value: "Unclassified", label: "Unclassified" },
  ];

  async function chooseDefaultRange(value: string) {
    if (value === defaultRangeValue) return;
    await runRangeAction(
      () =>
        api.PUT("/api/club", {
          body: { default_range_id: value === "none" ? null : Number(value) },
        }),
      "Default range saved.",
    );
  }

  async function chooseUnclassifiedClass(value: string) {
    if (value === club.unclassifiedClass) return;
    error = "";
    message = "";
    const result = await api.PUT("/api/club", {
      body: { unclassified_class: value as "Master" | "Unclassified" },
    });
    if (!result.response.ok) {
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

<div class="mx-auto flex max-w-3xl flex-col gap-4">
  <h1 class="m-0 text-xl">Club settings</h1>

  {#if loaded}
    {#if error}<ErrorState title={error} />{/if}
    {#if message}<p class="m-0 text-success-fg" role="status">{message}</p>{/if}

    <section
      aria-labelledby="club-name-heading"
      class="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 shadow-card"
    >
      <h2 id="club-name-heading" class="m-0 text-lg">Club name</h2>
      <form class="flex flex-wrap items-end gap-3" onsubmit={saveName}>
        <Field label="Name" class="min-w-56 flex-1">
          <Input bind:value={name} maxlength={100} />
        </Field>
        <Button type="submit">Save name</Button>
      </form>
    </section>

    <section
      aria-labelledby="club-class-heading"
      class="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 shadow-card"
    >
      <h2 id="club-class-heading" class="m-0 text-lg">Unclassified shooters</h2>
      <div class="flex flex-col gap-1">
        <span class="text-sm font-medium">Unclassified shooters compete as</span>
        <Select
          label="Unclassified shooters compete as"
          items={unclassifiedItems}
          value={club.unclassifiedClass}
          onValueChange={chooseUnclassifiedClass}
          class="sm:max-w-xs"
        />
      </div>
      <p class="m-0 text-sm text-muted">Applied when a match goes active (rule 20.2).</p>
    </section>

    <section
      aria-labelledby="club-ranges-heading"
      class="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 shadow-card"
    >
      <h2 id="club-ranges-heading" class="m-0 text-lg">Ranges</h2>
      <div class="flex flex-col gap-1">
        <span class="text-sm font-medium">Default range</span>
        <Select
          label="Default range"
          items={defaultRangeItems}
          value={defaultRangeValue}
          onValueChange={chooseDefaultRange}
          class="sm:max-w-xs"
        />
      </div>
      <p class="m-0 text-sm text-muted">
        Pre-filled on new competitions. A club with one active range selects it automatically.
      </p>
      {#if ranges.length === 0}
        <EmptyState title="No ranges yet." />
      {:else}
        <ul class="m-0 flex list-none flex-col divide-y divide-border p-0">
          {#each ranges as range (range.id)}
            <li aria-label={`Range ${range.name}`} class="py-2">
              {#if editingRange === range.id}
                <form
                  class="flex flex-wrap items-end gap-2"
                  onsubmit={(e) => renameRange(e, range)}
                >
                  <Field label="Range name" class="min-w-56 flex-1">
                    <Input bind:value={editRangeName} maxlength={100} required />
                  </Field>
                  <Button type="submit">Save</Button>
                  <Button variant="secondary" onclick={() => (editingRange = null)}>Cancel</Button>
                </form>
              {:else}
                <div class="flex flex-wrap items-center gap-2">
                  <span class="mr-auto font-medium">{range.name}</span>
                  {#if range.archived}<Badge>archived</Badge>{/if}
                  {#if range.id === club.defaultRangeId}<Badge tone="info">default</Badge>{/if}
                  <Button
                    variant="secondary"
                    size="sm"
                    onclick={() => startRename(range)}
                    aria-label={`Rename range ${range.name}`}>Rename</Button
                  >
                  {#if range.archived}
                    <Button
                      variant="secondary"
                      size="sm"
                      onclick={() => archiveRange(range, false)}
                      aria-label={`Restore range ${range.name}`}>Restore</Button
                    >
                  {:else if range.id !== club.defaultRangeId}
                    <Button
                      variant="secondary"
                      size="sm"
                      onclick={() => archiveRange(range, true)}
                      aria-label={`Archive range ${range.name}`}>Archive</Button
                    >
                  {/if}
                  {#if range.id !== club.defaultRangeId}
                    <Button
                      variant="danger"
                      size="sm"
                      onclick={() => (pendingDelete = range)}
                      aria-label={`Delete range ${range.name}`}>Delete</Button
                    >
                  {/if}
                </div>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
      <form class="flex flex-wrap items-end gap-3" onsubmit={addRange}>
        <Field label="Range to add" class="min-w-56 flex-1">
          <Input bind:value={newRangeName} maxlength={100} required />
        </Field>
        <Button type="submit">Add range</Button>
      </form>
    </section>

    <section
      aria-labelledby="club-logo-heading"
      class="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 shadow-card"
    >
      <h2 id="club-logo-heading" class="m-0 text-lg">Club logo</h2>
      {#if club.hasLogo}
        <img
          class="max-h-24 max-w-48 self-start rounded-md border border-border bg-surface-alt p-2"
          src={clubLogoUrl(club.version)}
          alt="Current club logo"
        />
        <div><Button variant="secondary" onclick={removeLogo}>Remove logo</Button></div>
      {:else}
        <p class="m-0 text-muted">No logo uploaded.</p>
      {/if}
      <form class="flex flex-wrap items-end gap-3" onsubmit={uploadLogo}>
        <Field label="Logo file" class="min-w-56 flex-1">
          <input
            type="file"
            accept="image/png,image/jpeg"
            bind:this={fileInput}
            class="min-h-11 w-full rounded-md border border-border bg-surface px-3 py-2 text-base file:mr-3 file:cursor-pointer file:rounded-md file:border-0 file:bg-brand-tint file:px-3 file:py-1 file:font-semibold file:text-link"
          />
        </Field>
        <Button type="submit">Upload logo</Button>
      </form>
      <p class="m-0 text-sm text-muted">PNG or JPEG, at most 512 KB and 2048 by 2048 pixels.</p>
    </section>
  {/if}
</div>

{#if pendingDelete}
  {@const target = pendingDelete}
  <ConfirmDialog
    title={`Delete range "${target.name}"?`}
    description="This permanently removes the range."
    confirmLabel="Delete"
    onconfirm={() => deleteRange(target)}
    oncancel={() => (pendingDelete = null)}
  />
{/if}
