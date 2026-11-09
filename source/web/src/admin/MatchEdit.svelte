<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { loadRulebook, rulebook } from "../lib/rulebook.svelte";
  import type { components } from "../lib/api/schema";
  import OfficialsInput from "./OfficialsInput.svelte";

  type Match = components["schemas"]["Match"];

  const datePattern = "\\d{4}-\\d{2}-\\d{2}";

  let {
    match,
    hasEntries,
    draft,
    saved,
    cancel,
  }: {
    match: Match;
    hasEntries: boolean;
    draft: boolean;
    saved: (match: Match) => void;
    cancel: () => void;
  } = $props();

  let name = $state(untrack(() => match.name));
  let date = $state(untrack(() => match.date));
  let course = $state(untrack(() => match.course));
  let classified = $state(untrack(() => match.classified));
  let refereeWaids = $state(untrack(() => match.referees.map((o) => o.waid)));
  let rangeOfficerWaids = $state(untrack(() => match.range_officers.map((o) => o.waid)));
  let error = $state("");

  onMount(loadRulebook);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const result = await api.PUT("/api/matches/{matchID}", {
      params: { path: { matchID: match.id } },
      body: {
        name,
        date,
        course,
        classified,
        referee_waids: refereeWaids.map((w) => w.trim()).filter(Boolean),
        range_officer_waids: rangeOfficerWaids.map((w) => w.trim()).filter(Boolean),
      },
    });
    if (result.data) saved(result.data);
    else error = failure(result);
  }
</script>

<form class="form-grid card" onsubmit={save} aria-label="Edit match">
  <label>Name <input bind:value={name} maxlength="128" required /></label>
  <label
    >Date <input
      type="text"
      bind:value={date}
      placeholder="YYYY-MM-DD"
      pattern={datePattern}
      inputmode="numeric"
      size="10"
      disabled={!draft}
      required
    /></label
  >
  <label>
    Course of fire
    <select bind:value={course} disabled={!draft || hasEntries}>
      {#each rulebook.courses as c (c.key)}<option value={c.key}>{c.label}</option>{/each}
    </select>
  </label>
  <label class="check">
    <input type="checkbox" bind:checked={classified} /> Classified match
  </label>
  <p class="muted">Leave a role empty to use the competition's officials.</p>
  <OfficialsInput role="Referee" bind:waids={refereeWaids} />
  <OfficialsInput role="Range officer" bind:waids={rangeOfficerWaids} />
  {#if !draft}<p class="muted">
      Only the name, Classified match and officials can change once the match has started.
    </p>
  {:else if hasEntries}<p class="muted">
      The course cannot change once the match has entries.
    </p>{/if}
  <button type="submit" class="primary">Save</button>
  <button type="button" onclick={cancel}>Cancel</button>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
</form>
