<script lang="ts">
  import { autoScroll } from "../lib/autoScroll";
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
  import { spectatorColumns, standingsGroups, type StandingRow } from "../lib/standings";
  import { SpectatorFeed } from "../lib/spectatorFeed.svelte";

  const feed = new SpectatorFeed();
  const control = feed.control;

  const columns = $derived(spectatorColumns(feed.standings));

  $effect(() => {
    void loadRulebook();
    void feed.loadMatches();
  });

  $effect(() => feed.startScreen());

  $effect(() => feed.watch(feed.viewed));

  const groups = $derived(
    standingsGroups(feed.standings, {
      groupByCategory: feed.snapshot?.group_by_category ?? false,
      categoryOrder: courseCategories(feed.current?.course ?? ""),
    }),
  );
  const headingHidden = $derived(groups.length <= 1 && groups[0]?.category === null);

  function rank(rows: StandingRow[], i: number): number {
    const tied = (a: StandingRow, b: StandingRow) =>
      a.total === b.total && (!feed.snapshot?.group_by_category || a.x_count === b.x_count);
    let r = i;
    while (r > 0 && tied(rows[r - 1], rows[i])) r--;
    return r + 1;
  }
</script>

{#snippet standingsTable(rows: StandingRow[], label: string)}
  <Table {label} class="[&_td]:py-2 [&_th]:text-sm">
    <thead>
      <tr>
        <th class="text-right">#</th>
        <th class="text-right">Lane</th>
        {#if columns.names}<th>Name</th>{/if}
        {#if columns.class}<th class="hidden md:table-cell">Classification</th>{/if}
        {#if columns.clubs}<th class="hidden md:table-cell">Club</th>{/if}
        {#if columns.countries}<th class="hidden md:table-cell">Country</th>{/if}
        <th class="hidden md:table-cell">Category</th>
        {#each columns.segments as n (n)}<th
            class="hidden text-right sm:table-cell"
            title={segmentLabel(feed.current?.course ?? "", n)}
            >{segmentColumn(feed.current?.course ?? "", n)}</th
          >{/each}
        <th class="text-right">Total</th>
        {#if columns.x}<th class="text-right">X</th>{/if}
        {#if feed.finished}<th>Scorecard</th>{/if}
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
          <td class="text-right text-lg font-bold tabular-nums"
            >{s.withdrawn ? "" : rank(rows, i)}</td
          >
          <td class="text-right tabular-nums">{s.lane}</td>
          {#if columns.names}<td class="whitespace-nowrap"
              >{s.first_name} {s.last_name}{s.withdrawn ? " (withdrawn)" : ""}</td
            >{/if}
          {#if columns.class}<td class="hidden whitespace-nowrap md:table-cell"
              >{s.classification && s.classification.value > 0
                ? s.classification.name
                : "Unclassified"}</td
            >{/if}
          {#if columns.clubs}<td class="hidden whitespace-nowrap md:table-cell">{s.club}</td>{/if}
          {#if columns.countries}<td class="hidden md:table-cell">{s.country}</td>{/if}
          <td class="hidden whitespace-nowrap md:table-cell">{s.category}</td>
          {#each columns.segments as n (n)}
            <td class="hidden text-right sm:table-cell"
              ><SegmentCell segment={s.segments?.find((st) => st.segment === n)} /></td
            >
          {/each}
          <td class="text-right text-lg font-bold tabular-nums"
            ><span class="sr-only">{s.total}</span><NumberFlow
              aria-hidden="true"
              value={s.total}
              format={{ useGrouping: false }}
            /></td
          >
          {#if columns.x}<td class="text-right tabular-nums"
              ><span class="sr-only">{s.x_count}</span><NumberFlow
                aria-hidden="true"
                value={s.x_count ?? 0}
                format={{ useGrouping: false }}
              /></td
            >{/if}
          {#if feed.finished}<td
              >{#if !s.withdrawn}<a
                  class="text-link underline"
                  href={`/scorecard/${feed.viewed}/${s.lane}`}
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

<main class="mx-auto flex w-full max-w-480 flex-col gap-4 p-3 sm:p-4 min-[90rem]:p-6">
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
  {#if control.managed && feed.viewed === null}
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
  {:else if !control.managed && feed.matches.length === 0}
    <EmptyState
      title="No matches are live yet."
      description="Reload this page once a match has started."
    />
  {:else}
    <div class="flex flex-wrap items-stretch gap-3">
      {#if !control.managed}
        <label
          class="flex min-w-0 max-w-full flex-none items-center gap-3 rounded-lg border border-border bg-surface px-4 py-2 font-semibold"
        >
          <span class="shrink-0">Match</span>
          <NativeSelect bind:value={feed.selected} class="w-80 max-w-full font-normal">
            {#each feed.matches as c (c.id)}
              <option value={c.id}>{c.name} ({c.date}, {c.status})</option>
            {/each}
          </NativeSelect>
        </label>
      {/if}
      <div
        class="flex flex-1 basis-80 flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-lg border border-border bg-surface px-4 py-3"
        role="status"
      >
        <div class="flex flex-wrap items-center gap-2">
          {#if feed.current}<Badge tone="neutral">{feed.current.status}</Badge>{/if}
          {#if feed.finished}
            <strong class="text-lg">Finished</strong>
            <a class="text-link underline" href={`/results/${feed.viewed}`}>View results</a>
          {:else if feed.snapshot && feed.acceptance}
            <strong class="text-lg"
              >Segment {feed.snapshot.current_segment} of {feed.snapshot.total_segments}</strong
            >
            <span class="text-muted"
              >{feed.acceptance.accepted}/{feed.acceptance.total} accepted</span
            >
          {/if}
        </div>
        <div class="flex flex-wrap items-center gap-2">
          {#if feed.connection === "stale"}
            <Badge tone="danger">Offline</Badge>
            {#if feed.updated}<span class="text-muted">Last update {clockTime(feed.updated)}</span
              >{/if}
          {:else if feed.updated}
            <Badge tone="success">Live</Badge>
            <span class="text-muted">Updated {clockTime(feed.updated)}</span>
          {:else}
            <Badge tone="neutral">Loading</Badge>
          {/if}
        </div>
      </div>
    </div>

    <h2 class={headingHidden ? "sr-only" : "m-0 text-lg"}>Standings</h2>
    {#if feed.standings.length === 0}
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
      {#if columns.segments.length > 0}
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
