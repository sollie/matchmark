<script lang="ts">
  import { untrack } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { localDate } from "../lib/datetime";
  import type { components } from "../lib/api/schema";

  type Competition = components["schemas"]["Competition"];

  const datePattern = "\\d{4}-\\d{2}-\\d{2}";

  let {
    competition,
    saved,
    cancel,
  }: {
    competition?: Competition;
    saved: () => void;
    cancel: () => void;
  } = $props();

  const today = localDate(new Date());
  let name = $state(untrack(() => competition?.name ?? ""));
  let startDate = $state(untrack(() => competition?.start_date ?? today));
  let endDate = $state(untrack(() => competition?.end_date ?? today));
  let error = $state("");

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const body = { name, start_date: startDate, end_date: endDate };
    const result = competition
      ? await api.PUT("/api/competitions/{competitionID}", { params: { path: { competitionID: competition.id } }, body })
      : await api.POST("/api/competitions", { body });
    if (result.error) error = failure(result);
    else saved();
  }
</script>

<form class="row card" onsubmit={save} aria-label={competition ? "Edit competition" : "New competition"}>
  <label>Competition name <input bind:value={name} maxlength="128" required /></label>
  <label>Start date <input type="text" bind:value={startDate} placeholder="YYYY-MM-DD" pattern={datePattern} inputmode="numeric" size="10" required /></label>
  <label>End date <input type="text" bind:value={endDate} placeholder="YYYY-MM-DD" pattern={datePattern} inputmode="numeric" size="10" required /></label>
  <button type="submit" class="primary">Save</button>
  <button type="button" onclick={cancel}>Cancel</button>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
</form>
