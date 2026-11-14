<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import type { components } from "../lib/api/schema";
  import { normalizeFirstName, normalizeLastName } from "../lib/names";
  import Button from "../lib/ui/Button.svelte";
  import Dialog from "../lib/ui/Dialog.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import LoadingState from "../lib/ui/LoadingState.svelte";
  import NameFields from "./NameFields.svelte";

  type Shooter = components["schemas"]["Shooter"];
  type ShooterInput = components["schemas"]["ShooterInput"];
  type Hit = components["schemas"]["Wa1500Shooter"];
  type ImportExtra = { update_shooter_id?: number; create_new?: boolean };

  let {
    query = "",
    onsaved,
    oncancel,
  }: {
    query?: string;
    onsaved: (shooter: Shooter) => void;
    oncancel?: () => void;
  } = $props();

  const blank = (): ShooterInput => ({
    last_name: "",
    first_name: "",
    country: "",
    club: "",
    waid: "",
  });

  let text = $state(query);
  let hits = $state<Hit[]>([]);
  let searched = $state(false);
  let searching = $state(false);
  let error = $state("");
  let manual = $state(false);
  let form = $state<ShooterInput>(blank());
  let pending = $state<{ hit: Hit; candidates: Shooter[] } | null>(null);
  let busy = $state(false);
  let archivedMatch = $state<{ hit: Hit; shooter: Shooter; extra: ImportExtra } | null>(null);

  const hitName = (h: Hit) => `${h.first_name} ${h.last_name}`;

  let searchForm: HTMLFormElement | undefined;

  onMount(() => {
    searchForm?.querySelector("input")?.focus();
    if (text.trim()) runSearch();
  });

  function search(e: SubmitEvent) {
    e.preventDefault();
    runSearch();
  }

  async function runSearch() {
    const q = text.trim();
    if (!q) return;
    error = "";
    searching = true;
    const result = await api.GET("/api/wa1500/search", { params: { query: { q } } });
    searching = false;
    searched = true;
    if (result.data) {
      hits = result.data.results;
      return;
    }
    hits = [];
    error = failure(result);
    if (result.response.status === 503) manual = true;
  }

  async function finish(id: number) {
    const result = await api.GET("/api/shooters/{shooterID}", {
      params: { path: { shooterID: id } },
    });
    if (result.data) onsaved(result.data);
    else error = failure(result);
  }

  async function runImport(hit: Hit, extra: ImportExtra = {}) {
    error = "";
    archivedMatch = null;
    busy = true;
    const result = await api.POST("/api/shooters/import", { body: { ...hit, ...extra } });
    busy = false;
    if (result.response.status === 409 && result.error && "candidates" in result.error) {
      pending = { hit, candidates: result.error.candidates };
      return;
    }
    if (result.response.status === 409 && result.error && "shooter" in result.error) {
      pending = null;
      archivedMatch = { hit, shooter: result.error.shooter, extra };
      return;
    }
    if (!result.data) {
      pending = null;
      error = failure(result);
      return;
    }
    pending = null;
    await finish(result.data.id);
  }

  async function unarchiveAndImport() {
    if (!archivedMatch) return;
    const { hit, shooter, extra } = archivedMatch;
    busy = true;
    const result = await api.POST("/api/shooters/{shooterID}/unarchive", {
      params: { path: { shooterID: shooter.id } },
    });
    busy = false;
    if (!result.response.ok) {
      error = failure(result);
      return;
    }
    await runImport(hit, extra);
  }

  async function saveManual(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    busy = true;
    const body = {
      ...form,
      first_name: normalizeFirstName(form.first_name),
      last_name: normalizeLastName(form.last_name),
    };
    const result = await api.POST("/api/shooters", { body });
    busy = false;
    if (!result.data) {
      error = failure(result);
      return;
    }
    await finish(result.data.id);
  }
</script>

<div class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card">
  <form class="flex flex-wrap items-end gap-3" onsubmit={search} bind:this={searchForm}>
    <Field label="Search WA1500" class="min-w-64 flex-1">
      <Input type="text" bind:value={text} maxlength={100} placeholder="Name, club or WAID" />
    </Field>
    <Button type="submit" disabled={searching || !text.trim()}>Search</Button>
    {#if oncancel}<Button variant="secondary" onclick={oncancel}>Close</Button>{/if}
  </form>

  {#if error}<ErrorState title={error} />{/if}

  {#if searching}
    <LoadingState label="Searching WA1500" />
  {:else if searched && hits.length === 0 && !error}
    <p class="m-0 text-muted" role="status">No WA1500 results</p>
  {/if}

  {#if hits.length > 0}
    <ul class="m-0 flex list-none flex-col gap-2 p-0" aria-label="WA1500 results">
      {#each hits as h (h.waid)}
        <li
          class="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
        >
          <span>{hitName(h)} · {h.waid} · {h.club} · {h.country}</span>
          <Button size="sm" disabled={busy} onclick={() => runImport(h)}>
            Import {hitName(h)}
          </Button>
        </li>
      {/each}
    </ul>
  {/if}

  {#if !manual}
    <div>
      <Button variant="secondary" onclick={() => (manual = true)}>Enter manually</Button>
    </div>
  {:else}
    <form
      class="flex flex-wrap items-end gap-3"
      onsubmit={saveManual}
      aria-label="Manual shooter entry"
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
      <Button type="submit" disabled={busy}>Save shooter</Button>
    </form>
  {/if}

  {#if archivedMatch}
    <div
      class="flex flex-col gap-3 rounded-lg border border-warning-fg bg-warning-bg p-4 text-warning-fg"
      role="alert"
    >
      <p class="m-0">
        {hitName(archivedMatch.hit)} matched an archived shooter ({archivedMatch.shooter.first_name}
        {archivedMatch.shooter.last_name}). Unarchive the shooter to import.
      </p>
      <div class="flex flex-wrap gap-2">
        <Button disabled={busy} onclick={unarchiveAndImport}>Unarchive</Button>
        <Button variant="secondary" onclick={() => (archivedMatch = null)}>Cancel</Button>
      </div>
    </div>
  {/if}

  {#if pending}
    {@const current = pending}
    <Dialog title="Name already exists" onclose={() => (pending = null)}>
      <div class="flex flex-col gap-3">
        <p class="m-0">
          {hitName(current.hit)} matches existing shooters. Update one of them with the WA1500 data, or
          create a new shooter.
        </p>
        <ul class="m-0 flex list-none flex-col gap-2 p-0">
          {#each current.candidates as c (c.id)}
            <li>
              <Button
                variant="secondary"
                class="h-auto w-full justify-start py-2 text-left whitespace-normal"
                disabled={busy}
                onclick={() => runImport(current.hit, { update_shooter_id: c.id })}
              >
                Update {c.first_name}
                {c.last_name} · {c.waid || "no WAID"} · {c.club} · {c.country}
              </Button>
            </li>
          {/each}
        </ul>
        <div class="flex flex-wrap gap-2">
          <Button disabled={busy} onclick={() => runImport(current.hit, { create_new: true })}>
            Create new shooter
          </Button>
          <Button variant="secondary" onclick={() => (pending = null)}>Cancel</Button>
        </div>
      </div>
    </Dialog>
  {/if}
</div>
