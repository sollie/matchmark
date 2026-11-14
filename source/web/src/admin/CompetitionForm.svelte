<script lang="ts">
  import { untrack } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { localDate } from "../lib/datetime";
  import { onMount } from "svelte";
  import { club, refreshClub } from "../lib/club.svelte";
  import { admin } from "../lib/session.svelte";
  import { dateError, dateRangeError, requiredError } from "../lib/formErrors";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Select from "../lib/ui/Select.svelte";
  import OfficialsInput from "./OfficialsInput.svelte";
  import type { components } from "../lib/api/schema";

  type Competition = components["schemas"]["Competition"];
  type Range = components["schemas"]["Range"];

  const noRange = "none";

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
  let refereeWaids = $state(
    untrack(() => competition?.referees.map((o) => o.waid) ?? (admin.waid ? [admin.waid] : [])),
  );
  let rangeOfficerWaids = $state(
    untrack(() => competition?.range_officers.map((o) => o.waid) ?? []),
  );
  let error = $state("");
  let submitted = $state(false);
  let ranges = $state<Range[]>([]);
  let rangeChoice = $state(
    untrack(() => (competition?.range_id == null ? noRange : String(competition.range_id))),
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
  const nameError = $derived(submitted ? requiredError(name, "a competition name") : "");
  const startError = $derived(submitted ? dateError(startDate) : "");
  const endError = $derived(
    submitted ? dateError(endDate) || dateRangeError(startDate, endDate) : "",
  );

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    submitted = true;
    if (
      requiredError(name, "a competition name") ||
      dateError(startDate) ||
      dateError(endDate) ||
      dateRangeError(startDate, endDate)
    )
      return;
    const body = {
      name: name.trim(),
      start_date: startDate.trim(),
      end_date: endDate.trim(),
      range_id: rangeChoice === noRange ? null : Number(rangeChoice),
      referee_waids: refereeWaids.map((w) => w.trim()).filter(Boolean),
      range_officer_waids: rangeOfficerWaids.map((w) => w.trim()).filter(Boolean),
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
  class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card"
  onsubmit={save}
  novalidate
  aria-label={competition ? "Edit competition" : "New competition"}
>
  <div class="grid gap-4 sm:grid-cols-2">
    <Field label="Competition name" error={nameError} class="sm:col-span-2">
      <Input bind:value={name} maxlength={128} aria-invalid={nameError ? true : undefined} />
    </Field>
    <Field label="Start date" error={startError}>
      <Input
        bind:value={startDate}
        placeholder="YYYY-MM-DD"
        inputmode="numeric"
        aria-invalid={startError ? true : undefined}
      />
    </Field>
    <Field label="End date" error={endError}>
      <Input
        bind:value={endDate}
        placeholder="YYYY-MM-DD"
        inputmode="numeric"
        aria-invalid={endError ? true : undefined}
      />
    </Field>
    <Field label="Range" class="sm:col-span-2">
      <Select
        label="Range"
        bind:value={rangeChoice}
        items={[
          { value: noRange, label: "No range" },
          ...pickable.map((r) => ({
            value: String(r.id),
            label: `${r.name}${r.archived ? " (archived)" : ""}`,
          })),
        ]}
      />
    </Field>
  </div>
  <OfficialsInput role="Referee" bind:waids={refereeWaids} />
  <OfficialsInput role="Range officer" bind:waids={rangeOfficerWaids} />
  <div class="flex flex-wrap gap-2">
    <Button type="submit">Save</Button>
    <Button variant="secondary" onclick={cancel}>Cancel</Button>
  </div>
  {#if error}<ErrorState title={error} />{/if}
</form>
