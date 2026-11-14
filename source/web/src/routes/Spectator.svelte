<script lang="ts">
  import { autoScroll } from "../lib/autoScroll";
  import { api } from "../lib/api/client";
  import { clockTime } from "../lib/datetime";
  import NumberFlow from "@number-flow/svelte";
  import Badge from "../lib/ui/Badge.svelte";
  import EmptyState from "../lib/ui/EmptyState.svelte";
  import NativeSelect from "../lib/ui/NativeSelect.svelte";
  import Table from "../lib/ui/Table.svelte";
  import SegmentCell from "../lib/SegmentCell.svelte";
  import {
    courseCategories,
    loadRulebook,
    segmentColumn,
    segmentLabel,
  } from "../lib/rulebook.svelte";
  import { openStream, type StreamState } from "../lib/sse";
  import { screenControl } from "../lib/screenControl.svelte";
  import type { components } from "../lib/api/schema";

  type Match = components["schemas"]["Match"];
  type Standing = components["schemas"]["Standing"];
  type Snapshot = components["schemas"]["SpectatorSnapshot"];

  let matches = $state<Match[]>([]);
  let selected = $state<number | null>(null);
  let standings = $state<Standing[]>([]);
  let snapshot = $state<Snapshot | null>(null);
  let connection = $state<StreamState>("connecting");
  let updated = $state<Date | null>(null);
  const control = screenControl;

  const viewed = $derived(
    control.managed
      ? control.assignment?.state === "match"
        ? control.assignment.match_id
        : null
      : selected,
  );

  const showNames = $derived(standings.some((s) => s.last_name !== undefined));
  const showClubs = $derived(standings.some((s) => s.club !== undefined));
  const showCountries = $derived(standings.some((s) => s.country !== undefined));
  const showClass = $derived(standings.some((s) => (s.classification?.value ?? 0) > 0));
  const showX = $derived(standings.some((s) => s.x_count !== undefined));
  const current = $derived(matches.find((c) => c.id === viewed));
  const finished = $derived((current?.status ?? snapshot?.status) === "finished");
  const acceptance = $derived.by(() => {
    if (!snapshot) return null;
    const segments = snapshot.standings
      .filter((s) => !s.withdrawn)
      .flatMap((s) => s.segments?.filter((st) => st.segment === snapshot?.current_segment) ?? []);
    return { accepted: segments.filter((st) => st.accepted).length, total: segments.length };
  });
  const segmentNumbers = $derived(
    [...new Set(standings.flatMap((s) => s.segments?.map((st) => st.segment) ?? []))].sort(
      (a, b) => a - b,
    ),
  );

  async function loadMatches() {
    const { data } = await api.GET("/api/spectator/matches");
    if (!data) return;
    matches = data.matches;
    if (control.managed) return;
    if (selected === null || !data.matches.some((c) => c.id === selected)) {
      selected = (data.matches.find((c) => c.status === "active") ?? data.matches[0])?.id ?? null;
    }
  }

  $effect(() => {
    void loadRulebook();
    void loadMatches();
  });

  $effect(() => {
    control.start();
    return () => control.stop();
  });

  $effect(() => {
    const id = viewed;
    snapshot = null;
    standings = [];
    updated = null;
    if (id === null) return;
    const stream = openStream({
      url: `/api/spectator/matches/${id}/stream`,
      onEvent: (event, data) => {
        if (event !== "snapshot") return;
        snapshot = JSON.parse(data);
        standings = snapshot?.standings ?? [];
        updated = new Date();
        const status = snapshot?.status;
        if (status) matches = matches.map((c) => (c.id === id ? { ...c, status } : c));
      },
      onState: (next) => {
        connection = next;
        if (next === "live") void loadMatches();
      },
    });
    return () => stream.close();
  });

  type Group = { category: string | null; classGroup?: string; rows: Standing[] };

  const withdrawnLast = (rows: Standing[]) =>
    [...rows].sort((a, b) => Number(a.withdrawn) - Number(b.withdrawn));

  const groups = $derived.by((): Group[] => {
    if (!snapshot?.group_by_category) return [{ category: null, rows: withdrawnLast(standings) }];
    const order = courseCategories(current?.course ?? "");
    const present = [...new Set(standings.map((s) => s.category))];
    const known = order.filter((c) => present.includes(c));
    const categories = [...known, ...present.filter((c) => !known.includes(c))];
    const byScore = (a: Standing, b: Standing) =>
      Number(a.withdrawn) - Number(b.withdrawn) ||
      b.total - a.total ||
      (b.x_count ?? 0) - (a.x_count ?? 0);
    const result = categories.flatMap((category): Group[] => {
      const rows = standings.filter((s) => s.category === category).sort(byScore);
      if (rows.some((s) => s.class_group === undefined)) return [{ category, rows }];
      const classes = [...new Set(rows.map((s) => s.class_group!.value))].sort((a, b) => b - a);
      return classes.map((value) => {
        const members = rows.filter((s) => s.class_group!.value === value);
        return { category, classGroup: members[0].class_group!.name, rows: members };
      });
    });
    if (categories.length <= 1 && result.every((g) => g.classGroup === undefined)) {
      return [{ category: null, rows: withdrawnLast(standings) }];
    }
    return result;
  });
  const headingHidden = $derived(groups.length <= 1 && groups[0]?.category === null);

  function rank(rows: Standing[], i: number): number {
    const tied = (a: Standing, b: Standing) =>
      a.total === b.total && (!snapshot?.group_by_category || a.x_count === b.x_count);
    let r = i;
    while (r > 0 && tied(rows[r - 1], rows[i])) r--;
    return r + 1;
  }
</script>

{#snippet standingsTable(rows: Standing[], label: string)}
  <Table
    {label}
    class="min-[90rem]:text-xl [&_td]:py-3 [&_th]:text-sm min-[90rem]:[&_th]:text-base"
  >
    <thead>
      <tr>
        <th class="text-right">#</th>
        <th class="text-right">Lane</th>
        {#if showNames}<th>Name</th>{/if}
        {#if showClass}<th class="hidden md:table-cell">Classification</th>{/if}
        {#if showClubs}<th class="hidden md:table-cell">Club</th>{/if}
        {#if showCountries}<th class="hidden md:table-cell">Country</th>{/if}
        <th class="hidden md:table-cell">Category</th>
        {#each segmentNumbers as n (n)}<th
            class="hidden text-right sm:table-cell"
            title={segmentLabel(current?.course ?? "", n)}
            >{segmentColumn(current?.course ?? "", n)}</th
          >{/each}
        <th class="text-right">Total</th>
        {#if showX}<th class="text-right">X</th>{/if}
        {#if finished}<th>Scorecard</th>{/if}
      </tr>
    </thead>
    <tbody>
      {#each rows as s, i (s.lane)}
        {@const leader = !s.withdrawn && rank(rows, i) === 1}
        <tr
          class={[
            s.withdrawn && "text-muted",
            leader && "font-semibold [&>td]:bg-brand-tint",
            !leader && "[&>td]:bg-surface",
          ]}
        >
          <td class="text-right text-lg font-bold tabular-nums min-[90rem]:text-2xl"
            >{s.withdrawn ? "" : rank(rows, i)}</td
          >
          <td class="text-right tabular-nums">{s.lane}</td>
          {#if showNames}<td>{s.first_name} {s.last_name}{s.withdrawn ? " (withdrawn)" : ""}</td
            >{/if}
          {#if showClass}<td class="hidden md:table-cell"
              >{s.classification && s.classification.value > 0
                ? s.classification.name
                : "Unclassified"}</td
            >{/if}
          {#if showClubs}<td class="hidden md:table-cell">{s.club}</td>{/if}
          {#if showCountries}<td class="hidden md:table-cell">{s.country}</td>{/if}
          <td class="hidden md:table-cell">{s.category}</td>
          {#each segmentNumbers as n (n)}
            <td class="hidden text-right sm:table-cell"
              ><SegmentCell segment={s.segments?.find((st) => st.segment === n)} /></td
            >
          {/each}
          <td class="text-right text-lg font-bold tabular-nums min-[90rem]:text-2xl"
            ><span class="sr-only">{s.total}</span><NumberFlow
              aria-hidden="true"
              value={s.total}
              format={{ useGrouping: false }}
            /></td
          >
          {#if showX}<td class="text-right tabular-nums"
              ><span class="sr-only">{s.x_count}</span><NumberFlow
                aria-hidden="true"
                value={s.x_count ?? 0}
                format={{ useGrouping: false }}
              /></td
            >{/if}
          {#if finished}<td
              >{#if !s.withdrawn}<a
                  class="text-link underline"
                  href={`/scorecard/${viewed}/${s.lane}`}
                  aria-label={`Scorecard for lane ${s.lane}`}>View</a
                >{/if}</td
            >{/if}
        </tr>
      {/each}
    </tbody>
  </Table>
{/snippet}

{#snippet legendItem(
  segment: {
    segment: number;
    accepted: boolean;
    state: "not_started" | "in_progress" | "awaiting_acceptance" | "accepted";
    score?: number;
  },
  text: string,
)}
  <li class="flex items-center gap-2">
    <SegmentCell {segment} prefix={false} />
    {text}
  </li>
{/snippet}

<main
  class="mx-auto flex w-full max-w-480 flex-col gap-4 p-3 sm:p-4 min-[90rem]:gap-6 min-[90rem]:p-8 min-[90rem]:text-lg"
>
  <h1 class="sr-only">Live standings</h1>
  {#if control.notice || control.error}
    <p
      class="m-0 rounded-lg border border-border bg-danger-bg px-4 py-3 text-danger-fg"
      role="alert"
    >
      {control.notice || control.error}
    </p>
  {/if}
  {#if control.managed && control.registration && !control.assignment?.approved}
    <div
      class="flex flex-col items-center gap-2 rounded-xl border-4 border-brand bg-brand-tint px-4 py-8 text-center"
      role="region"
      aria-label="Remote control"
    >
      <span class="text-xl font-semibold">Remote control code</span>
      <strong
        class="code font-mono text-7xl font-extrabold tracking-[0.15em] text-fg sm:text-8xl min-[90rem]:text-9xl"
        >{control.registration.code}</strong
      >
      <span class="text-base text-muted"
        >Enter this code in the admin Screens page to approve this screen.</span
      >
    </div>
  {/if}
  {#if control.managed && viewed === null}
    <div
      class="flex min-h-[40dvh] flex-col items-center justify-center gap-3 rounded-xl border border-border bg-surface px-4 py-12 text-center"
      role="status"
    >
      <h2 class="m-0 text-5xl font-extrabold sm:text-6xl min-[90rem]:text-8xl">
        Waiting for a match
      </h2>
      <p class="m-0 text-xl text-muted min-[90rem]:text-3xl">
        This screen will show a match as soon as one is assigned.
      </p>
    </div>
  {:else if !control.managed && matches.length === 0}
    <EmptyState
      title="No matches are live yet."
      description="Reload this page once a match has started."
    />
  {:else}
    <div class="flex flex-wrap items-stretch gap-3">
      {#if !control.managed}
        <label
          class="flex min-w-0 max-w-full flex-1 items-center gap-3 rounded-lg border border-border bg-surface px-4 py-2 font-semibold"
        >
          Match
          <NativeSelect bind:value={selected} class="max-w-full flex-1 font-normal">
            {#each matches as c (c.id)}
              <option value={c.id}>{c.name} ({c.date}, {c.status})</option>
            {/each}
          </NativeSelect>
        </label>
      {/if}
      <div
        class="flex flex-[100_1_14rem] flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-lg border border-border bg-surface px-4 py-3"
        role="status"
      >
        <div class="flex flex-wrap items-center gap-2">
          {#if current}<Badge tone="neutral">{current.status}</Badge>{/if}
          {#if finished}
            <strong class="text-lg">Finished</strong>
            <a class="text-link underline" href={`/results/${viewed}`}>View results</a>
          {:else if snapshot && acceptance}
            <strong class="text-lg"
              >Segment {snapshot.current_segment} of {snapshot.total_segments}</strong
            >
            <span class="text-muted">{acceptance.accepted}/{acceptance.total} accepted</span>
          {/if}
        </div>
        <div class="flex flex-wrap items-center gap-2">
          {#if connection === "stale"}
            <Badge tone="danger">Offline</Badge>
            {#if updated}<span class="text-muted">Last update {clockTime(updated)}</span>{/if}
          {:else if updated}
            <Badge tone="success">Live</Badge>
            <span class="text-muted">Updated {clockTime(updated)}</span>
          {:else}
            <Badge tone="neutral">Loading</Badge>
          {/if}
        </div>
      </div>
    </div>

    <h2 class={headingHidden ? "sr-only" : "m-0 text-lg"}>Standings</h2>
    {#if standings.length === 0}
      <EmptyState title="No results yet." heading="h3" />
    {:else}
      <div class="flex min-h-0 flex-col gap-4 overflow-y-auto" use:autoScroll>
        {#each groups as group (`${group.category ?? ""}/${group.classGroup ?? ""}`)}
          {#if group.category !== null}<h3 class="m-0 text-lg">
              {group.classGroup ? `${group.category} - ${group.classGroup}` : group.category}
            </h3>{/if}
          {@render standingsTable(
            group.rows,
            group.category === null
              ? "Standings"
              : group.classGroup
                ? `${group.category} - ${group.classGroup}`
                : group.category,
          )}
        {/each}
      </div>
      {#if segmentNumbers.length > 0}
        <ul
          class="m-0 flex list-none flex-wrap gap-x-4 gap-y-2 p-0 text-sm text-muted"
          aria-label="Segment state legend"
        >
          {@render legendItem({ segment: 0, accepted: false, state: "not_started" }, "Not started")}
          {@render legendItem({ segment: 0, accepted: false, state: "in_progress" }, "In progress")}
          {@render legendItem(
            { segment: 0, accepted: false, state: "awaiting_acceptance" },
            "Awaiting acceptance",
          )}
          {@render legendItem(
            { segment: 0, accepted: true, state: "accepted", score: 118 },
            "Accepted",
          )}
        </ul>
      {/if}
    {/if}
  {/if}
</main>
