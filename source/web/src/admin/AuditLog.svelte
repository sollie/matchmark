<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
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

<h1>Audit log</h1>
{#if error}<p class="error" role="alert">{error}</p>{/if}

<form class="row card" onsubmit={search}>
  <label>
    User
    <input bind:value={user} autocomplete="off" />
  </label>
  <label>
    Action
    <input bind:value={action} placeholder="e.g. segment.lock" autocomplete="off" />
  </label>
  <label>
    From
    <input type="date" bind:value={from} />
  </label>
  <label>
    To
    <input type="date" bind:value={to} />
  </label>
  <button type="submit" class="primary">Filter</button>
</form>
<p class="muted">Entries are kept forever and cannot be edited or deleted. Dates are UTC.</p>

<div class="card scroll">
  <table>
    <thead>
      <tr><th>Time</th><th>User</th><th>Action</th><th>Target</th><th>Details</th><th>IP</th></tr>
    </thead>
    <tbody>
      {#each entries as e (e.id)}
        <tr>
          <td>{new Date(e.time).toLocaleString()}</td>
          <td>{e.username}</td>
          <td>{e.action}</td>
          <td>{e.target}</td>
          <td>{e.details}</td>
          <td>{e.ip}</td>
        </tr>
      {:else}
        <tr><td colspan="6" class="muted">No entries match.</td></tr>
      {/each}
    </tbody>
  </table>
</div>

<div class="row">
  <button type="button" disabled={page <= 1} onclick={() => go(page - 1)}>Previous</button>
  <span>Page {page} of {pages} ({total} entries)</span>
  <button type="button" disabled={page >= pages} onclick={() => go(page + 1)}>Next</button>
</div>
