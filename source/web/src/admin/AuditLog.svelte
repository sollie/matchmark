<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Table from "../lib/ui/Table.svelte";
  import type { components } from "../lib/api/schema";

  type Entry = components["schemas"]["AuditEntry"];

  const pageSize = 25;

  let entries = $state<Entry[]>([]);
  let total = $state(0);
  let page = $state(1);
  let user = $state("");
  let action = $state("");
  let from = $state("");
  let to = $state("");
  let error = $state("");

  const pages = $derived(Math.max(1, Math.ceil(total / pageSize)));

  async function load() {
    error = "";
    const result = await api.GET("/api/audit-log", {
      params: {
        query: {
          page,
          page_size: pageSize,
          user: user.trim() || undefined,
          action: action.trim() || undefined,
          from: from || undefined,
          to: to || undefined,
        },
      },
    });
    if (result.data) {
      entries = result.data.entries;
      total = result.data.total;
    } else {
      error = failure(result);
    }
  }

  onMount(load);

  function search(e: SubmitEvent) {
    e.preventDefault();
    page = 1;
    return load();
  }

  function go(next: number) {
    page = next;
    return load();
  }
</script>

<div class="flex flex-col gap-4">
  <h1 class="m-0 text-xl">Audit log</h1>
  {#if error}<ErrorState title={error} />{/if}

  <form
    class="grid gap-4 rounded-xl border border-border bg-surface p-4 shadow-card sm:grid-cols-2 lg:grid-cols-[1fr_1fr_10rem_10rem_auto] lg:items-end"
    onsubmit={search}
    aria-label="Audit log filter"
  >
    <Field label="User"><Input bind:value={user} autocomplete="off" /></Field>
    <Field label="Action">
      <Input bind:value={action} placeholder="e.g. segment.lock" autocomplete="off" />
    </Field>
    <Field label="From"><Input type="date" bind:value={from} /></Field>
    <Field label="To"><Input type="date" bind:value={to} /></Field>
    <Button type="submit">Filter</Button>
  </form>
  <p class="m-0 text-sm text-muted">
    Entries are kept forever and cannot be edited or deleted. Dates are UTC.
  </p>

  <Table label="Audit log entries" tableLabel="Audit log">
    <thead>
      <tr><th>Time</th><th>User</th><th>Action</th><th>Target</th><th>Details</th><th>IP</th></tr>
    </thead>
    <tbody>
      {#each entries as e (e.id)}
        <tr>
          <td class="whitespace-nowrap">{new Date(e.time).toLocaleString()}</td>
          <td>{e.username}</td>
          <td class="font-mono text-sm">{e.action}</td>
          <td>{e.target}</td>
          <td>{e.details}</td>
          <td class="font-mono text-sm">{e.ip}</td>
        </tr>
      {:else}
        <tr><td colspan="6" class="text-muted">No entries match.</td></tr>
      {/each}
    </tbody>
  </Table>

  <div class="flex flex-wrap items-center gap-3">
    <Button variant="secondary" disabled={page <= 1} onclick={() => go(page - 1)}>Previous</Button>
    <span>Page {page} of {pages} ({total} entries)</span>
    <Button variant="secondary" disabled={page >= pages} onclick={() => go(page + 1)}>Next</Button>
  </div>
</div>
