import type { components } from "./api/schema";

export type StandingRow = components["schemas"]["Standing"];
export type ResultsCategory = components["schemas"]["Results"]["categories"][number];
type ResultsRow = components["schemas"]["ResultsRow"];

export type SpectatorColumns = {
  names: boolean;
  clubs: boolean;
  countries: boolean;
  class: boolean;
  x: boolean;
  segments: number[];
};

export type ResultsColumns = {
  names: boolean;
  clubs: boolean;
  class: boolean;
  countries: boolean;
  remarks: boolean;
  segments: boolean;
  x: boolean;
};

export type StandingsGroup = {
  category: string | null;
  classGroup?: string;
  rows: StandingRow[];
};

type Visible = {
  last_name?: unknown;
  club?: unknown;
  country?: unknown;
  classification?: { value: number };
  x_count?: unknown;
};

const flags = {
  names: (r: Visible) => r.last_name !== undefined,
  clubs: (r: Visible) => r.club !== undefined,
  countries: (r: Visible) => r.country !== undefined,
  class: (r: Visible) => (r.classification?.value ?? 0) > 0,
  x: (r: Visible) => r.x_count !== undefined,
};

export function spectatorColumns(standings: StandingRow[]): SpectatorColumns {
  return {
    names: standings.some(flags.names),
    clubs: standings.some(flags.clubs),
    countries: standings.some(flags.countries),
    class: standings.some(flags.class),
    x: standings.some(flags.x),
    segments: [
      ...new Set(standings.flatMap((s) => s.segments?.map((st) => st.segment) ?? [])),
    ].sort((a, b) => a - b),
  };
}

const allRows = (c: ResultsCategory): ResultsRow[] => [
  ...c.rows,
  ...c.groups.flatMap((g) => g.rows),
];

export function resultsColumns(categories: ResultsCategory[]): ResultsColumns {
  const any = (test: (r: ResultsRow) => boolean) => categories.some((c) => allRows(c).some(test));
  return {
    names: any(flags.names),
    clubs: any(flags.clubs),
    class: any(flags.class),
    countries: any(flags.countries),
    remarks: any((r) => r.remarks !== undefined),
    segments: any((r) => r.segments.some((s) => s.score !== undefined)),
    x: any(flags.x),
  };
}

const withdrawnLast = (rows: StandingRow[]) =>
  rows.toSorted((a, b) => Number(a.withdrawn) - Number(b.withdrawn));

export function standingsGroups(
  standings: StandingRow[],
  options: { groupByCategory: boolean; categoryOrder: string[] },
): StandingsGroup[] {
  if (!options.groupByCategory) return [{ category: null, rows: withdrawnLast(standings) }];
  const present = [...new Set(standings.map((s) => s.category))];
  const known = options.categoryOrder.filter((c) => present.includes(c));
  const categories = [...known, ...present.filter((c) => !known.includes(c))];
  const byScore = (a: StandingRow, b: StandingRow) =>
    Number(a.withdrawn) - Number(b.withdrawn) ||
    b.total - a.total ||
    (b.x_count ?? 0) - (a.x_count ?? 0);
  const result = categories.flatMap((category): StandingsGroup[] => {
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
}
