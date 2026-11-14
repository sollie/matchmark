<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api/client";
  import { failure } from "../lib/api/util";
  import { navigate } from "../lib/router.svelte";
  import { statusTone } from "../lib/domain";
  import { courseLabel, is1500, loadRulebook, rulebook } from "../lib/rulebook.svelte";
  import { localDate } from "../lib/datetime";
  import { dateError, dateRangeError, requiredError } from "../lib/formErrors";
  import Badge from "../lib/ui/Badge.svelte";
  import Button, { buttonVariants } from "../lib/ui/Button.svelte";
  import ConfirmDialog from "../lib/ui/ConfirmDialog.svelte";
  import EmptyState from "../lib/ui/EmptyState.svelte";
  import ErrorState from "../lib/ui/ErrorState.svelte";
  import Field from "../lib/ui/Field.svelte";
  import Input from "../lib/ui/Input.svelte";
  import Select from "../lib/ui/Select.svelte";
  import Table from "../lib/ui/Table.svelte";
  import Download from "@lucide/svelte/icons/download";
  import CompetitionForm from "./CompetitionForm.svelte";
  import OfficialsInput from "./OfficialsInput.svelte";
  import type { components } from "../lib/api/schema";

  type Match = components["schemas"]["Match"];
  type Competition = components["schemas"]["Competition"];

  const sameDay = "oneday";
  const newCompetition = "new";
  const downloadFormats = [
    { format: "xlsx", label: "XLSX zip", title: "Results reports (.xlsx) zip" },
    { format: "csv", label: "CSV zip", title: "Detailed CSVs zip" },
    { format: "pdf", label: "PDF zip", title: "Results PDFs zip" },
    { format: "all", label: "Download all", title: "All formats zip" },
  ];
  const matchDownloads = [
    { label: "XLSX", path: "results.xlsx", title: "Results report (.xlsx)" },
    { label: "CSV", path: "results.csv", title: "Detailed CSV" },
    { label: "PDF", path: "pdf", title: "Results PDF" },
  ];

  let matches = $state<Match[]>([]);
  let competitions = $state<Competition[]>([]);
  let loaded = $state(false);
  let competitionChoice = $state(sameDay);
  let compName = $state("");
  let compStart = $state(localDate(new Date()));
  let compEnd = $state(localDate(new Date()));
  let creatingCompetition = $state(false);
  let editingCompetition = $state<number | null>(null);
  let deletingCompetition = $state<Competition | null>(null);
  let name = $state("");
  let date = $state(localDate(new Date()));
  let course = $state("150");
  let refereeWaids = $state<string[]>([]);
  let rangeOfficerWaids = $state<string[]>([]);
  let error = $state("");
  let submitted = $state(false);

  const creatingNew = $derived(competitionChoice === newCompetition);
  const nameError = $derived(submitted ? requiredError(name, "a match name") : "");
  const dateIssue = $derived(submitted ? dateError(date) : "");
  const compNameError = $derived(
    submitted && creatingNew ? requiredError(compName, "a competition name") : "",
  );
  const compStartError = $derived(submitted && creatingNew ? dateError(compStart) : "");
  const compEndError = $derived(
    submitted && creatingNew ? dateError(compEnd) || dateRangeError(compStart, compEnd) : "",
  );

  async function load() {
    const [m, c] = await Promise.all([api.GET("/api/matches"), api.GET("/api/competitions")]);
    if (m.data) matches = m.data.matches;
    else error = failure(m);
    if (c.data) competitions = c.data.competitions;
    else error = failure(c);
    loaded = true;
  }

  const groups = $derived(
    competitions.map((comp) => {
      const inComp = matches.filter((m) => m.competition_id === comp.id);
      const matches1500 = inComp.filter((m) => is1500(m.course));
      return {
        comp,
        matches: inComp,
        hasFinished1500: matches1500.some((m) => m.status === "finished"),
        open1500: matches1500.filter((m) => m.status !== "finished"),
      };
    }),
  );

  async function competitionSaved() {
    creatingCompetition = false;
    editingCompetition = null;
    await load();
  }

  async function removeCompetition(comp: Competition) {
    error = "";
    const result = await api.DELETE("/api/competitions/{competitionID}", {
      params: { path: { competitionID: comp.id } },
    });
    if (result.error) error = failure(result);
    else await load();
  }

  onMount(load);
  onMount(loadRulebook);

  async function create(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    submitted = true;
    if (
      requiredError(name, "a match name") ||
      dateError(date) ||
      (creatingNew &&
        (requiredError(compName, "a competition name") ||
          dateError(compStart) ||
          dateError(compEnd) ||
          dateRangeError(compStart, compEnd)))
    )
      return;
    const body: components["schemas"]["MatchInput"] = {
      name: name.trim(),
      date: date.trim(),
      course,
      referee_waids: refereeWaids.map((w) => w.trim()).filter(Boolean),
      range_officer_waids: rangeOfficerWaids.map((w) => w.trim()).filter(Boolean),
    };
    if (creatingNew) {
      body.competition = {
        name: compName.trim(),
        start_date: compStart.trim(),
        end_date: compEnd.trim(),
      };
    } else if (competitionChoice !== sameDay) {
      body.competition_id = Number(competitionChoice);
    }
    const result = await api.POST("/api/matches", { body });
    if (result.data) navigate(`/admin/matches/${result.data.id}`);
    else error = failure(result);
  }
</script>

<div class="flex flex-col gap-6">
  <h1 class="m-0 text-xl">Matches</h1>
  {#if error}<ErrorState title={error} />{/if}

  <form
    class="flex flex-col gap-4 rounded-xl border border-border bg-surface p-4 shadow-card"
    onsubmit={create}
    novalidate
    aria-label="Create match"
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
          aria-invalid={dateIssue ? true : undefined}
        />
      </Field>
      <Field label="Course of fire">
        <Select
          label="Course of fire"
          bind:value={course}
          items={rulebook.courses.map((c) => ({ value: c.key, label: c.label }))}
        />
      </Field>
      <Field label="Competition" class="sm:col-span-2">
        <Select
          label="Competition"
          bind:value={competitionChoice}
          items={[
            { value: sameDay, label: "New one-day competition" },
            ...competitions.map((comp) => ({
              value: String(comp.id),
              label: `${comp.name} (${comp.start_date} – ${comp.end_date})`,
            })),
            { value: newCompetition, label: "New competition…" },
          ]}
        />
      </Field>
      {#if creatingNew}
        <Field label="Competition name" error={compNameError} class="sm:col-span-2">
          <Input
            bind:value={compName}
            maxlength={128}
            aria-invalid={compNameError ? true : undefined}
          />
        </Field>
        <Field label="Start date" error={compStartError}>
          <Input
            bind:value={compStart}
            placeholder="YYYY-MM-DD"
            inputmode="numeric"
            aria-invalid={compStartError ? true : undefined}
          />
        </Field>
        <Field label="End date" error={compEndError}>
          <Input
            bind:value={compEnd}
            placeholder="YYYY-MM-DD"
            inputmode="numeric"
            aria-invalid={compEndError ? true : undefined}
          />
        </Field>
      {/if}
    </div>
    <p class="m-0 text-sm text-muted">Leave a role empty to use the competition's officials.</p>
    <OfficialsInput role="Referee" bind:waids={refereeWaids} />
    <OfficialsInput role="Range officer" bind:waids={rangeOfficerWaids} />
    <div><Button type="submit">Create match</Button></div>
  </form>

  <div class="flex flex-wrap items-center justify-between gap-3">
    <h2 class="m-0 text-lg">Competitions</h2>
    {#if !creatingCompetition}
      <Button variant="secondary" onclick={() => (creatingCompetition = true)}
        >New competition</Button
      >
    {/if}
  </div>
  {#if creatingCompetition}
    <CompetitionForm saved={competitionSaved} cancel={() => (creatingCompetition = false)} />
  {/if}

  {#each groups as { comp, matches: inComp, hasFinished1500, open1500 } (comp.id)}
    <section
      class="flex flex-col gap-3 rounded-xl border border-border bg-surface-alt p-4"
      aria-label={`Competition ${comp.name}`}
    >
      <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
        <h3 class="m-0 text-lg">{comp.name}</h3>
        <span class="text-muted"
          >{comp.start_date === comp.end_date
            ? comp.start_date
            : `${comp.start_date} – ${comp.end_date}`}</span
        >
        {#if comp.range_name}<span class="text-muted">{comp.range_name}</span>{/if}
      </div>
      <div class="flex flex-wrap gap-2">
        <Button
          variant="secondary"
          size="sm"
          onclick={() => (editingCompetition = comp.id)}
          aria-label={`Edit competition ${comp.name}`}>Edit</Button
        >
        {#if inComp.length > 0}
          {#each downloadFormats as f (f.format)}
            <a
              class={buttonVariants({ variant: "secondary", size: "sm" })}
              href={`/api/competitions/${comp.id}/results.zip?format=${f.format}`}
              download
              title={f.title}
              aria-label={`${f.title} for ${comp.name}`}
              ><Download class="size-4" aria-hidden="true" />{f.label}</a
            >
          {/each}
        {/if}
        {#if hasFinished1500}
          <a
            class={buttonVariants({ variant: "secondary", size: "sm" })}
            href={`/api/competitions/${comp.id}/results.xlsx`}
            download
            title="Combined 1500 workbook (.xlsx)"
            aria-label={`Combined 1500 workbook (.xlsx) for ${comp.name}`}
            ><Download class="size-4" aria-hidden="true" />1500 XLSX</a
          >
        {/if}
        {#if inComp.length === 0}
          <Button
            variant="danger"
            size="sm"
            onclick={() => (deletingCompetition = comp)}
            aria-label={`Delete competition ${comp.name}`}>Delete</Button
          >
        {/if}
      </div>
      {#if hasFinished1500 && open1500.length > 0}
        <p class="m-0 text-sm text-muted" role="note">
          The 1500 XLSX leaves out unfinished 1500 matches: {open1500
            .map((m) => m.name)
            .join(", ")}.
        </p>
      {/if}
      {#if editingCompetition === comp.id}
        <CompetitionForm
          competition={comp}
          saved={competitionSaved}
          cancel={() => (editingCompetition = null)}
        />
      {/if}
      <Table label={`Matches in ${comp.name}`}>
        <thead>
          <tr><th>Name</th><th>Date</th><th>Course</th><th>Status</th><th>Actions</th></tr>
        </thead>
        <tbody>
          {#each inComp as c (c.id)}
            <tr>
              <td
                ><a class="font-semibold text-link" href={`/admin/matches/${c.id}`}>{c.name}</a></td
              >
              <td>{c.date}</td>
              <td>{courseLabel(c.course)}</td>
              <td><Badge tone={statusTone[c.status]}>{c.status}</Badge></td>
              <td>
                {#if c.status === "finished"}
                  <div class="flex flex-wrap gap-2">
                    {#each matchDownloads as d (d.label)}
                      <a
                        class={buttonVariants({ variant: "secondary", size: "sm" })}
                        href={d.path === "pdf"
                          ? `/api/admin/matches/${c.id}/results?format=pdf`
                          : `/api/matches/${c.id}/${d.path}`}
                        download
                        title={d.title}
                        aria-label={`${d.title} for ${c.name}`}
                        ><Download class="size-4" aria-hidden="true" />{d.label}</a
                      >
                    {/each}
                  </div>
                {/if}
              </td>
            </tr>
          {:else}
            <tr><td colspan="5" class="text-muted">No matches yet.</td></tr>
          {/each}
        </tbody>
      </Table>
    </section>
  {:else}
    {#if loaded}<EmptyState
        title="No competitions yet"
        description="Create a match above or add a competition to get started."
      />{/if}
  {/each}
</div>

{#if deletingCompetition}
  {@const comp = deletingCompetition}
  <ConfirmDialog
    title="Delete competition"
    description={`Delete competition "${comp.name}"? It has no matches. This cannot be undone.`}
    confirmLabel="Delete competition"
    onconfirm={async () => {
      const target = comp;
      deletingCompetition = null;
      await removeCompetition(target);
    }}
    oncancel={() => (deletingCompetition = null)}
  />
{/if}
