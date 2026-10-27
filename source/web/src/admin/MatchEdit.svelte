<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { loadRulebook, rulebook } from "../lib/rulebook.svelte";
  import type { components } from "../lib/api/schema";

  type Match = components["schemas"]["Match"];

  const datePattern = "\\d{4}-\\d{2}-\\d{2}";

  let {
    match,
    hasEntries,
    saved,
    cancel,
  }: {
    match: Match;
    hasEntries: boolean;
    saved: (match: Match) => void;
    cancel: () => void;
  } = $props();

  let name = $state(untrack(() => match.name));
  let date = $state(untrack(() => match.date));
  let course = $state(untrack(() => match.course));
  let error = $state("");

  onMount(loadRulebook);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const result = await api.PUT("/api/matches/{matchID}", {
      params: { path: { matchID: match.id } },
      body: { name, date, course },
    });
    if (result.data) saved(result.data);
    else error = failure(result);
  }
</script>

<form class="row card" onsubmit={save} aria-label="Edit match">
  <label>Name <input bind:value={name} maxlength="128" required /></label>
  <label
    >Date <input
      type="text"
      bind:value={date}
      placeholder="YYYY-MM-DD"
      pattern={datePattern}
      inputmode="numeric"
      size="10"
      required
    /></label
  >
  <label>
    Course of fire
    <select bind:value={course} disabled={hasEntries}>
      {#each rulebook.courses as c (c.key)}<option value={c.key}>{c.label}</option>{/each}
    </select>
  </label>
  {#if hasEntries}<p class="muted">The course cannot change once the match has entries.</p>{/if}
  <button type="submit" class="primary">Save</button>
  <button type="button" onclick={cancel}>Cancel</button>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
</form>
