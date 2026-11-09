<script lang="ts">
  import { untrack } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { localDate } from "../lib/datetime";
  import { onMount } from "svelte";
  import { club, refreshClub } from "../lib/club.svelte";
  import type { components } from "../lib/api/schema";

  type Competition = components["schemas"]["Competition"];
  type Range = components["schemas"]["Range"];

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
  let ranges = $state<Range[]>([]);
  let rangeChoice = $state(
    untrack(() => (competition?.range_id == null ? "" : String(competition.range_id))),
  );

  onMount(async () => {
    const { data } = await api.GET("/api/ranges");
    if (!data) return;
    ranges = data.ranges;
    if (competition) return;
    await refreshClub();
    const active = ranges.filter((r) => !r.archived);
    if (club.defaultRangeId !== null) rangeChoice = String(club.defaultRangeId);
    else if (active.length === 1) rangeChoice = String(active[0].id);
  });

  const pickable = $derived(ranges.filter((r) => !r.archived || r.id === competition?.range_id));

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    const body = {
      name,
      start_date: startDate,
      end_date: endDate,
      range_id: rangeChoice === "" ? null : Number(rangeChoice),
    };
    const result = competition
      ? await api.PUT("/api/competitions/{competitionID}", {
          params: { path: { competitionID: competition.id } },
          body,
        })
      : await api.POST("/api/competitions", { body });
    if (result.error) error = failure(result);
    else saved();
  }
</script>

<form
  class="row card"
  onsubmit={save}
  aria-label={competition ? "Edit competition" : "New competition"}
>
  <label>Competition name <input bind:value={name} maxlength="128" required /></label>
  <label
    >Start date <input
      type="text"
      bind:value={startDate}
      placeholder="YYYY-MM-DD"
      pattern={datePattern}
      inputmode="numeric"
      size="10"
      required
    /></label
  >
  <label
    >End date <input
      type="text"
      bind:value={endDate}
      placeholder="YYYY-MM-DD"
      pattern={datePattern}
      inputmode="numeric"
      size="10"
      required
    /></label
  >
  <label
    >Range
    <select bind:value={rangeChoice}>
      <option value="">No range</option>
      {#each pickable as range (range.id)}
        <option value={String(range.id)}>{range.name}{range.archived ? " (archived)" : ""}</option>
      {/each}
    </select></label
  >
  <button type="submit" class="primary">Save</button>
  <button type="button" onclick={cancel}>Cancel</button>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
</form>
