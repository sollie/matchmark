<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import Wa1500Import from "./Wa1500Import.svelte";
  import NameFields from "./NameFields.svelte";
  import ShooterMerge from "./ShooterMerge.svelte";
  import Badge from "../lib/ui/Badge.svelte";
  import Button from "../lib/ui/Button.svelte";
  import Dialog from "../lib/ui/Dialog.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Table from "../lib/ui/Table.svelte";
  import ClassChips from "../lib/ClassChips.svelte";
  import { normalizeFirstName, normalizeLastName } from "../lib/names";

  type Shooter = components["schemas"]["Shooter"];
  type ShooterListItem = components["schemas"]["ShooterListItem"];
  type ShooterMatch = components["schemas"]["ShooterDetail"]["matches"][number];
  type ShooterInput = components["schemas"]["ShooterInput"];

  const blank = (): ShooterInput => ({
    last_name: "",
    first_name: "",
    country: "NOR",
    club: "",
    waid: "",
  });

  let shooters = $state<ShooterListItem[]>([]);
  let expanded = $state<number | null>(null);
  let matches = $state<ShooterMatch[]>([]);
  let form = $state<ShooterInput>(blank());
  let editing = $state<number | null>(null);
  let error = $state("");
  let query = $state("");
  let importing = $state(false);
  let notice = $state("");
  let merging = $state<ShooterListItem | null>(null);
  let blocked = $state<{ shooter: ShooterListItem; matches: ShooterMatch[] } | null>(null);
  let timer: ReturnType<typeof setTimeout> | undefined;

  async function load() {
    const q = query.trim();
    const result = await api.GET("/api/shooters", {
      params: { query: q ? { q, limit: 100, include_archived: true } : {} },
    });
    if (result.data) shooters = result.data.shooters;
    else error = failure(result);
  }

  function onenter(e: KeyboardEvent) {
    if (e.key !== "Enter") return;
    clearTimeout(timer);
    load();
  }

  function onsearch() {
    clearTimeout(timer);
    timer = setTimeout(load, 250);
  }

  async function imported(s: Shooter) {
    importing = false;
    query = "";
    notice = `Saved ${s.first_name} ${s.last_name}.`;
    await load();
  }

  async function merged(target: Shooter) {
    const source = merging;
    merging = null;
    notice = `Merged ${source?.first_name} ${source?.last_name} into ${target.first_name} ${target.last_name}.`;
    expanded = null;
    await load();
  }

  onMount(load);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const body = {
      ...form,
      first_name: normalizeFirstName(form.first_name),
      last_name: normalizeLastName(form.last_name),
    };
    const result =
      editing === null
        ? await api.POST("/api/shooters", { body })
        : await api.PUT("/api/shooters/{shooterID}", {
            params: { path: { shooterID: editing } },
            body,
          });
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    cancel();
    await load();
  }

  async function fetchMatches(s: ShooterListItem) {
    const result = await api.GET("/api/shooters/{shooterID}", {
      params: { path: { shooterID: s.id } },
    });
    if (!result.data) {
      error = failure(result);
      return null;
    }
    return result.data.matches;
  }

  async function toggleMatches(s: ShooterListItem) {
    if (expanded === s.id) {
      expanded = null;
      return;
    }
    const loaded = await fetchMatches(s);
    if (!loaded) return;
    matches = loaded;
    expanded = s.id;
  }

  async function explainBlocked(s: ShooterListItem) {
    const loaded = await fetchMatches(s);
    if (loaded) blocked = { shooter: s, matches: loaded };
  }

  const entriesLabel = (n: number) => `Entered in ${n} ${n === 1 ? "match" : "matches"}`;

  function edit(s: Shooter) {
    editing = s.id;
    form = {
      last_name: s.last_name,
      first_name: s.first_name,
      country: s.country,
      club: s.club,
      waid: s.waid,
    };
  }

  function cancel() {
    editing = null;
    form = blank();
  }

  async function setArchived(s: ShooterListItem, archive: boolean) {
    error = "";
    const result = await (archive
      ? api.POST("/api/shooters/{shooterID}/archive", { params: { path: { shooterID: s.id } } })
      : api.POST("/api/shooters/{shooterID}/unarchive", { params: { path: { shooterID: s.id } } }));
    if (!result.response.ok) error = failure(result);
    await load();
  }

  async function remove(s: ShooterListItem) {
    if (!confirm(`Delete ${s.first_name} ${s.last_name}?`)) return;
    error = "";
    const result = await api.DELETE("/api/shooters/{shooterID}", {
      params: { path: { shooterID: s.id } },
    });
    if (!result.response.ok) error = failure(result);
    await load();
  }
</script>

<div class="flex flex-col gap-4">
  <h1 class="m-0 text-xl">Shooters</h1>
  {#if error}<ErrorState title={error} />{/if}
  {#if notice}<p class="m-0 text-success-fg" role="status">{notice}</p>{/if}

  <div class="flex flex-wrap items-end gap-3">
    <Field label="Search shooters" class="min-w-64 flex-1">
      <Input
        type="search"
        bind:value={query}
        oninput={onsearch}
        onkeydown={onenter}
        placeholder="Name or WAID"
      />
    </Field>
    <Button variant="secondary" aria-expanded={importing} onclick={() => (importing = !importing)}>
      Import from WA1500
    </Button>
  </div>

  {#if merging}
    <ShooterMerge source={merging} onmerged={merged} oncancel={() => (merging = null)} />
  {/if}

  {#if blocked}
    {@const b = blocked}
    {@const finished = b.matches.filter((m) => m.status === "finished")}
    <Dialog
      title={`Can't delete ${b.shooter.first_name} ${b.shooter.last_name}`}
      onclose={() => (blocked = null)}
    >
      <div class="flex flex-col gap-3">
        <p class="m-0">
          {b.shooter.first_name}
          {b.shooter.last_name} is entered in these matches:
        </p>
        <ul class="m-0 list-disc pl-5">
          {#each b.matches as m (m.id)}
            <li>
              <a class="text-link" href={`/admin/matches/${m.id}`}>{m.name}</a>, {m.date}, {m.status}
            </li>
          {/each}
        </ul>
        {#if finished.length > 0}
          <p class="m-0">
            Entries in finished matches cannot be removed, so archive or merge this shooter instead.
          </p>
        {:else}
          <p class="m-0">
            Remove the entries from these matches first, or archive or merge this shooter instead.
          </p>
        {/if}
        <div class="flex flex-wrap gap-2">
          <Button
            variant="secondary"
            onclick={() => {
              const s = b.shooter;
              blocked = null;
              setArchived(s, !s.archived);
            }}>{b.shooter.archived ? "Unarchive" : "Archive"}</Button
          >
          <Button
            variant="secondary"
            onclick={() => {
              merging = b.shooter;
              blocked = null;
            }}>Merge into…</Button
          >
          <Button onclick={() => (blocked = null)}>Close</Button>
        </div>
      </div>
    </Dialog>
  {/if}

  {#if importing}
    <Wa1500Import onsaved={imported} oncancel={() => (importing = false)} />
  {/if}

  <form
    class="flex flex-wrap items-end gap-3 rounded-xl border border-border bg-surface p-4 shadow-card"
    onsubmit={save}
  >
    <NameFields bind:first={form.first_name} bind:last={form.last_name} />
    <Field label="Country" class="w-24">
      <Input bind:value={form.country} maxlength={3} required />
    </Field>
    <Field label="Club">
      <Input bind:value={form.club} maxlength={128} required />
    </Field>
    <Field label="WAID (blank for new shooter)">
      <Input bind:value={form.waid} maxlength={32} />
    </Field>
    <Button type="submit">{editing === null ? "Add shooter" : "Save"}</Button>
    {#if editing !== null}<Button variant="secondary" onclick={cancel}>Cancel</Button>{/if}
  </form>

  <Table label="Shooters list">
    <thead>
      <tr><th>Name</th><th>Country</th><th>Club</th><th>WAID</th><th>Entries</th><th></th></tr>
    </thead>
    <tbody>
      {#each shooters as s (s.id)}
        <tr>
          <td>
            <div class="flex flex-col items-start gap-1">
              <span class="font-semibold">
                {s.first_name}
                {s.last_name}
              </span>
              {#if s.archived}<Badge>Archived</Badge>{/if}
              <ClassChips classifications={s.classifications} />
            </div>
          </td>
          <td>{s.country}</td>
          <td>{s.club}</td>
          <td>{s.waid || "—"}</td>
          <td>
            {#if s.entry_count > 0}
              <Button
                variant="secondary"
                size="sm"
                aria-expanded={expanded === s.id}
                onclick={() => toggleMatches(s)}>{entriesLabel(s.entry_count)}</Button
              >
              {#if expanded === s.id}
                <ul class="m-0 mt-2 list-disc pl-5">
                  {#each matches as m (m.id)}
                    <li>
                      <a class="text-link" href={`/admin/matches/${m.id}`}>{m.name}</a>, {m.date}, {m.status}
                    </li>
                  {/each}
                </ul>
              {/if}
            {:else}
              <span class="text-muted">None</span>
            {/if}
          </td>
          <td>
            <div class="flex flex-wrap gap-2">
              <Button variant="secondary" size="sm" onclick={() => edit(s)}>Edit</Button>
              {#if s.archived}
                <Button variant="secondary" size="sm" onclick={() => setArchived(s, false)}
                  >Unarchive</Button
                >
              {:else}
                <Button variant="secondary" size="sm" onclick={() => setArchived(s, true)}
                  >Archive</Button
                >
              {/if}
              <Button variant="secondary" size="sm" onclick={() => (merging = s)}
                >Merge into…</Button
              >
              <Button
                variant="danger"
                size="sm"
                aria-disabled={s.entry_count > 0 ? "true" : undefined}
                onclick={() => (s.entry_count > 0 ? explainBlocked(s) : remove(s))}>Delete</Button
              >
            </div>
          </td>
        </tr>
      {:else}
        <tr>
          <td colspan="6" class="text-muted">
            {query.trim() ? "No matching shooters." : "No shooters yet."}
          </td>
        </tr>
      {/each}
    </tbody>
  </Table>
</div>
