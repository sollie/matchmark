<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { loadRulebook, rulebook } from "../lib/rulebook.svelte";
  import { dateError, requiredError } from "../lib/formErrors";
  import type { components } from "../lib/api/schema";
  import Button from "../lib/ui/Button.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Select from "../lib/ui/Select.svelte";
  import OfficialsInput from "./OfficialsInput.svelte";

  type Match = components["schemas"]["Match"];

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
  let submitted = $state(false);

  const nameError = $derived(submitted ? requiredError(name, "a match name") : "");
  const dateIssue = $derived(submitted && draft ? dateError(date) : "");

  onMount(loadRulebook);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    submitted = true;
    if (requiredError(name, "a match name") || (draft && dateError(date))) return;
    const result = await api.PUT("/api/matches/{matchID}", {
      params: { path: { matchID: match.id } },
      body: {
        name: name.trim(),
        date: date.trim(),
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

<form
  class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card"
  onsubmit={save}
  novalidate
  aria-label="Edit match"
>
  <div class="grid gap-4 sm:grid-cols-2">
    <Field label="Name" error={nameError} class="sm:col-span-2">
      <Input bind:value={name} maxlength={128} aria-invalid={nameError ? true : undefined} />
    </Field>
    <Field label="Date" error={dateIssue}>
      <Input
        bind:value={date}
        placeholder="YYYY-MM-DD"
        inputmode="numeric"
        disabled={!draft}
        aria-invalid={dateIssue ? true : undefined}
      />
    </Field>
    <Field label="Course of fire">
      <Select
        label="Course of fire"
        bind:value={course}
        disabled={!draft || hasEntries}
        items={rulebook.courses.map((c) => ({ value: c.key, label: c.label }))}
      />
    </Field>
  </div>
  <label class="flex min-h-11 items-center gap-2">
    <input type="checkbox" class="size-5 accent-brand" bind:checked={classified} /> Classified match
  </label>
  <p class="m-0 text-sm text-muted">Leave a role empty to use the competition's officials.</p>
  <OfficialsInput role="Referee" bind:waids={refereeWaids} />
  <OfficialsInput role="Range officer" bind:waids={rangeOfficerWaids} />
  {#if !draft}
    <p class="m-0 text-sm text-muted">
      Only the name, Classified match and officials can change once the match has started.
    </p>
  {:else if hasEntries}
    <p class="m-0 text-sm text-muted">The course cannot change once the match has entries.</p>
  {/if}
  <div class="flex flex-wrap gap-2">
    <Button type="submit">Save</Button>
    <Button variant="secondary" onclick={cancel}>Cancel</Button>
  </div>
  {#if error}<ErrorState title={error} />{/if}
</form>
