<script lang="ts">
  import { api } from "../lib/api/client";
  import { brand } from "../lib/brand";
  import { club, clubLogoUrl } from "../lib/club.svelte";
  import { matchRoute, navigate, router } from "../lib/router.svelte";
  import { loadRulebook, segmentLabel } from "../lib/rulebook.svelte";
  import Button, { buttonVariants } from "../lib/ui/Button.svelte";
  import EmptyState from "../lib/ui/EmptyState.svelte";
  import LoadingState from "../lib/ui/LoadingState.svelte";
  import Select from "../lib/ui/Select.svelte";
  import Table from "../lib/ui/Table.svelte";
  import type { components } from "../lib/api/schema";

  type Match = components["schemas"]["Match"];
  type Results = components["schemas"]["Results"];
  type Row = components["schemas"]["ResultsRow"];

  const rings = [
    ["X", "x"],
    ["10", "10"],
    ["9", "9"],
    ["8", "8"],
    ["7", "7"],
    ["0", "0"],
    ["M", "miss"],
  ] as const;

  type Category = Results["categories"][number];

  const allRows = (c: Category): Row[] => [...c.rows, ...c.groups.flatMap((g) => g.rows)];

  function sections(c: Category) {
    if (c.groups.length === 0) return [{ heading: null, rows: c.rows, dnf: c.did_not_finish }];
    return c.groups.map((g, i) => ({
      heading: g.class.name,
      rows: g.rows,
      dnf: i === c.groups.length - 1 ? c.did_not_finish : [],
    }));
  }

  function hasCounts(row: Row): boolean {
    return row.segments.some((s) => s.ring_counts !== undefined);
  }

  function ringTotal(row: Row, key: (typeof rings)[number][1]): number {
    return row.segments.reduce((sum, s) => sum + (s.ring_counts?.[key] ?? 0), 0);
  }

  function cardTitle(row: Row): string {
    return row.last_name === undefined ? `#${row.rank}` : `${row.first_name} ${row.last_name}`;
  }

  let matches = $state<Match[]>([]);
  let loaded = $state(false);
  let results = $state<Results | null>(null);
  let missing = $state(false);

  const id = $derived(matchRoute("/results/:id", router.path)?.id ?? null);
  const showNames = $derived(
    results?.categories.some((c) => allRows(c).some((r) => r.last_name !== undefined)) ?? false,
  );
  const showClubs = $derived(
    results?.categories.some((c) => allRows(c).some((r) => r.club !== undefined)) ?? false,
  );
  const showClass = $derived(
    results?.categories.some((c) => allRows(c).some((r) => (r.classification?.value ?? 0) > 0)) ??
      false,
  );
  const showCountries = $derived(
    results?.categories.some((c) => allRows(c).some((r) => r.country !== undefined)) ?? false,
  );
  const showRemarks = $derived(
    results?.categories.some((c) => allRows(c).some((r) => r.remarks !== undefined)) ?? false,
  );
  const showSegments = $derived(
    results?.categories.some((c) =>
      allRows(c).some((r) => r.segments.some((s) => s.score !== undefined)),
    ) ?? false,
  );
  const showX = $derived(
    results?.categories.some((c) => allRows(c).some((r) => r.x_count !== undefined)) ?? false,
  );

  $effect(() => {
    void loadRulebook();
    void (async () => {
      const { data } = await api.GET("/api/spectator/matches");
      matches = (data?.matches ?? []).filter((c) => c.status === "finished");
      loaded = true;
    })();
  });

  $effect(() => {
    results = null;
    missing = false;
    if (id === null) return;
    const current = id;
    void (async () => {
      const { data, response } = await api.GET("/api/spectator/matches/{matchID}/results", {
        params: { path: { matchID: Number(current) } },
      });
      if (current !== id) return;
      if (data) results = data;
      else missing = response.status === 404;
    })();
  });

  async function printPdf() {
    if (!results) return;
    const response = await fetch(`/api/matches/${results.match.id}/results?format=pdf`);
    if (!response.ok) return;
    const url = URL.createObjectURL(await response.blob());
    const frame = document.createElement("iframe");
    frame.style.cssText = "position:fixed;width:0;height:0;border:0;visibility:hidden";
    frame.onload = () => {
      frame.contentWindow?.focus();
      frame.contentWindow?.print();
      setTimeout(() => {
        frame.remove();
        URL.revokeObjectURL(url);
      }, 60000);
    };
    frame.src = url;
    document.body.append(frame);
  }

  function pick(value: string) {
    if (value && value !== id) navigate(`/results/${value}`);
  }

  const options = $derived(
    matches.map((c) => ({ value: String(c.id), label: `${c.name} (${c.date})` })),
  );
</script>

<main
  class="mx-auto w-full max-w-5xl px-4 py-4 print:max-w-none print:p-0 min-[1440px]:max-w-none min-[1440px]:px-10 min-[1440px]:text-lg"
>
  <div class="flex flex-col gap-3 print:hidden">
    <h1 class="m-0 text-xl min-[1440px]:text-3xl">Results</h1>
    <div class="flex flex-wrap items-end gap-3">
      <div class="w-full max-w-md min-w-0 flex-1">
        <Select
          label="Finished match"
          value={id ?? ""}
          items={options}
          placeholder={loaded && matches.length === 0 ? "No finished matches" : "Select a match"}
          onValueChange={pick}
        />
      </div>
      {#if results}
        <Button variant="secondary" onclick={printPdf}>Print</Button>
        <a
          class={buttonVariants({ variant: "primary" })}
          href={`/api/matches/${results.match.id}/results?format=pdf`}
          download>Download PDF</a
        >
      {/if}
    </div>
  </div>

  {#if missing}
    <EmptyState class="mt-6" title="Results are not available for this match." />
  {:else if results}
    <article class="mt-4 print:mt-0">
      <header class="my-4 flex items-center gap-4">
        {#if club.hasLogo}
          <img
            class="block h-16 w-auto max-w-32 object-contain min-[1440px]:h-24 min-[1440px]:max-w-48"
            src={clubLogoUrl(club.version)}
            alt=""
          />
        {/if}
        <div>
          {#if club.name}<p class="m-0 font-semibold">{club.name}</p>{/if}
          <h2 class="m-0 text-xl min-[1440px]:text-3xl">{results.match.name}</h2>
          <p class="m-0 text-muted print:text-black">
            {results.match.date} · {results.match.course_label}
          </p>
        </div>
      </header>

      <div class="grid grid-cols-1 items-start gap-x-10 gap-y-6 min-[1440px]:grid-cols-2">
        {#each results.categories as category (category.category)}
          <section class="min-w-0 break-inside-avoid">
            <h3 class="m-0 mb-2 text-lg min-[1440px]:text-2xl">{category.category}</h3>
            {#each sections(category) as section (section.heading ?? "")}
              {#if section.heading !== null}
                <h4 class="m-0 mt-4 mb-2 text-base font-semibold text-muted">{section.heading}</h4>
              {/if}
              <Table label={`Standings, ${category.category}`} class="print:text-[10pt]">
                <thead>
                  <tr>
                    <th class="text-right">#</th>
                    {#if showNames}<th>Shooter</th><th>WAID</th>{/if}
                    {#if showClass}<th>Classification</th>{/if}
                    {#if showClubs}<th>Club</th>{/if}
                    {#if showCountries}<th>Country</th>{/if}
                    {#if showSegments}{#each results.segments as n (n)}<th class="text-right"
                          >{segmentLabel(results.match.course, n)}</th
                        >{/each}{/if}
                    <th class="text-right">Total</th>
                    {#if showX}<th class="text-right">X</th>{/if}
                    {#if showRemarks}<th>Remarks</th>{/if}
                  </tr>
                </thead>
                <tbody>
                  {#each section.rows as row, i (i)}
                    <tr>
                      <td class="text-right tabular-nums">{row.rank}</td>
                      {#if showNames}<td>{row.first_name} {row.last_name}</td><td
                          >{row.waid ?? ""}</td
                        >{/if}
                      {#if showClass}<td
                          >{row.classification && row.classification.value > 0
                            ? row.classification.name
                            : ""}</td
                        >{/if}
                      {#if showClubs}<td>{row.club}</td>{/if}
                      {#if showCountries}<td>{row.country}</td>{/if}
                      {#if showSegments}{#each row.segments as st (st.segment)}<td
                            class="text-right tabular-nums">{st.score ?? ""}</td
                          >{/each}{/if}
                      <td class="text-right font-bold tabular-nums">{row.total}</td>
                      {#if showX}<td class="text-right tabular-nums">{row.x_count}</td>{/if}
                      {#if showRemarks}<td>{row.remarks ?? ""}</td>{/if}
                    </tr>
                  {/each}
                  {#each section.dnf as row, i (i)}
                    <tr class="text-muted">
                      <td class="text-right"></td>
                      {#if showNames}<td>{row.first_name} {row.last_name}</td><td
                          >{row.waid ?? ""}</td
                        >{/if}
                      {#if showClass}<td></td>{/if}
                      {#if showClubs}<td>{row.club}</td>{/if}
                      {#if showCountries}<td>{row.country}</td>{/if}
                      {#if showSegments}{#each row.segments as st (st.segment)}<td
                            class="text-right tabular-nums">{st.score ?? ""}</td
                          >{/each}{/if}
                      <td class="text-right font-bold">DNF</td>
                      {#if showX}<td class="text-right"></td>{/if}
                      {#if showRemarks}<td>{row.remarks ?? ""}</td>{/if}
                    </tr>
                  {/each}
                </tbody>
              </Table>
              {#each section.rows.filter(hasCounts) as row (row.rank + cardTitle(row))}
                <div class="my-4 break-inside-avoid">
                  <Table
                    label={`Ring counts, ${cardTitle(row)}`}
                    tableLabel={`Score card for ${cardTitle(row)}`}
                  >
                    <thead>
                      <tr>
                        <th>{cardTitle(row)}</th>
                        {#each results.segments as n (n)}<th class="text-right"
                            >{segmentLabel(results.match.course, n)}</th
                          >{/each}
                        <th class="text-right">Total</th>
                      </tr>
                    </thead>
                    <tbody>
                      {#each rings as [label, key] (key)}
                        <tr>
                          <th scope="row" class="text-left">{label}</th>
                          {#each row.segments as st (st.segment)}<td class="text-right tabular-nums"
                              >{st.ring_counts?.[key] ?? ""}</td
                            >{/each}
                          <td class="text-right font-bold tabular-nums">{ringTotal(row, key)}</td>
                        </tr>
                      {/each}
                    </tbody>
                  </Table>
                </div>
              {/each}
            {/each}
          </section>
        {/each}

        {#if results.new_shooters.length > 0}
          <section class="min-w-0 break-inside-avoid">
            <h3 class="m-0 mb-2 text-lg min-[1440px]:text-2xl">New shooters</h3>
            <Table label="New shooters" class="print:text-[10pt]">
              <thead>
                <tr>
                  <th>Shooter</th>
                  <th>Category</th>
                  {#if showClubs}<th>Club</th>{/if}
                  {#if showCountries}<th>Country</th>{/if}
                  <th class="text-right">Total</th>
                  {#if showX}<th class="text-right">X</th>{/if}
                </tr>
              </thead>
              <tbody>
                {#each results.new_shooters as s, i (i)}
                  <tr>
                    <td>{s.first_name} {s.last_name}</td>
                    <td>{s.category}</td>
                    {#if showClubs}<td>{s.club}</td>{/if}
                    {#if showCountries}<td>{s.country}</td>{/if}
                    <td class="text-right font-bold tabular-nums">{s.total}</td>
                    {#if showX}<td class="text-right tabular-nums">{s.x_count}</td>{/if}
                  </tr>
                {/each}
              </tbody>
            </Table>
          </section>
        {/if}
      </div>

      {#if results.categories.length === 0}
        <p class="text-muted">No results.</p>
      {/if}
    </article>
  {:else if id !== null}
    <LoadingState label="Loading results" />
  {/if}
  <p class="print:hidden"><a href="/">Live standings</a></p>
  <p class="mt-4 hidden text-[9pt] print:block">
    Scored with MatchMark · <a href={brand.repoUrl}>{brand.repoUrl}</a>
  </p>
</main>
