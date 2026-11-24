import { describe, expect, it } from "vitest";
import {
  resultsColumns,
  spectatorColumns,
  standingsGroups,
  type ResultsCategory,
  type StandingRow,
} from "./standings";

const row = (over: Partial<StandingRow> = {}): StandingRow =>
  ({ lane: 1, category: "Open", total: 0, withdrawn: false, ...over }) as StandingRow;

describe("spectatorColumns", () => {
  it("hides every optional column when no row carries the field", () => {
    expect(spectatorColumns([row()])).toEqual({
      names: false,
      clubs: false,
      countries: false,
      class: false,
      x: false,
      segments: [],
    });
  });

  it("shows a column when any row carries the field", () => {
    const columns = spectatorColumns([
      row(),
      row({ last_name: "Doe", club: "A", country: "SE", x_count: 0 }),
    ]);
    expect(columns).toMatchObject({ names: true, clubs: true, countries: true, x: true });
  });

  it("shows class only for a positive classification value", () => {
    expect(
      spectatorColumns([row({ classification: { value: 0, name: "Unclassified" } })]).class,
    ).toBe(false);
    expect(spectatorColumns([row({ classification: { value: 2, name: "Expert" } })]).class).toBe(
      true,
    );
  });

  it("lists distinct segment numbers in ascending order", () => {
    const segment = (n: number) => ({ segment: n, accepted: false, state: "in_progress" as const });
    const columns = spectatorColumns([
      row({ segments: [segment(3), segment(1)] }),
      row({ segments: [segment(2), segment(1)] }),
    ]);
    expect(columns.segments).toEqual([1, 2, 3]);
  });
});

describe("resultsColumns", () => {
  const result = (over: Record<string, unknown> = {}) =>
    ({ rank: 1, total: 0, segments: [], ...over }) as unknown as ResultsCategory["rows"][number];
  const category = (
    rows: ResultsCategory["rows"],
    groupRows: ResultsCategory["rows"] = [],
  ): ResultsCategory =>
    ({
      category: "Open",
      rows,
      groups:
        groupRows.length > 0 ? [{ class: { value: 1, name: "Marksman" }, rows: groupRows }] : [],
      did_not_finish: [],
    }) as unknown as ResultsCategory;

  it("hides everything without categories", () => {
    expect(resultsColumns([])).toEqual({
      names: false,
      clubs: false,
      class: false,
      countries: false,
      remarks: false,
      segments: false,
      x: false,
    });
  });

  it("looks at rows inside class groups too", () => {
    const columns = resultsColumns([
      category([result()], [result({ remarks: "PB", x_count: 4, last_name: "Doe" })]),
    ]);
    expect(columns).toMatchObject({ remarks: true, x: true, names: true });
  });

  it("shows segments only when some segment has a score", () => {
    const empty = result({ segments: [{ segment: 1 }] });
    const scored = result({ segments: [{ segment: 1, score: 90 }] });
    expect(resultsColumns([category([empty])]).segments).toBe(false);
    expect(resultsColumns([category([empty, scored])]).segments).toBe(true);
  });
});

describe("standingsGroups", () => {
  const order = ["Open", "Junior"];

  it("returns one headingless group with withdrawn rows last when not grouped", () => {
    const rows = [
      row({ lane: 1, withdrawn: true, total: 99 }),
      row({ lane: 2, total: 10 }),
      row({ lane: 3, total: 50 }),
    ];
    const groups = standingsGroups(rows, { groupByCategory: false, categoryOrder: order });
    expect(groups).toHaveLength(1);
    expect(groups[0].category).toBeNull();
    expect(groups[0].rows.map((r) => r.lane)).toEqual([2, 3, 1]);
  });

  it("orders categories by the rulebook and appends unknown ones", () => {
    const rows = [
      row({ lane: 1, category: "Zed" }),
      row({ lane: 2, category: "Junior" }),
      row({ lane: 3, category: "Open" }),
      row({ lane: 4, category: "Veteran" }),
    ];
    const groups = standingsGroups(rows, { groupByCategory: true, categoryOrder: order });
    expect(groups.map((g) => g.category)).toEqual(["Open", "Junior", "Zed", "Veteran"]);
  });

  it("sorts by withdrawn, then total, then X within a category", () => {
    const rows = [
      row({ lane: 1, total: 100, x_count: 2 }),
      row({ lane: 2, total: 100, x_count: 5 }),
      row({ lane: 3, total: 120, withdrawn: true }),
      row({ lane: 4, total: 110 }),
      row({ lane: 5, category: "Junior" }),
    ];
    const groups = standingsGroups(rows, { groupByCategory: true, categoryOrder: order });
    expect(groups[0].rows.map((r) => r.lane)).toEqual([4, 2, 1, 3]);
  });

  it("splits a category into class groups, highest value first", () => {
    const cg = (value: number, name: "Marksman" | "Master") => ({ value, name });
    const rows = [
      row({ lane: 1, class_group: cg(1, "Marksman") }),
      row({ lane: 2, class_group: cg(3, "Master") }),
      row({ lane: 3, class_group: cg(1, "Marksman") }),
      row({ lane: 4, category: "Junior", class_group: cg(3, "Master") }),
    ];
    const groups = standingsGroups(rows, { groupByCategory: true, categoryOrder: order });
    expect(groups.map((g) => [g.category, g.classGroup, g.rows.map((r) => r.lane)])).toEqual([
      ["Open", "Master", [2]],
      ["Open", "Marksman", [1, 3]],
      ["Junior", "Master", [4]],
    ]);
  });

  it("keeps a category whole when any row lacks a class group", () => {
    const rows = [
      row({ lane: 1, class_group: { value: 1, name: "Marksman" } }),
      row({ lane: 2 }),
      row({ lane: 3, category: "Junior" }),
    ];
    const groups = standingsGroups(rows, { groupByCategory: true, categoryOrder: order });
    expect(groups.map((g) => [g.category, g.classGroup])).toEqual([
      ["Open", undefined],
      ["Junior", undefined],
    ]);
  });

  it("collapses to the combined view for a single category without class groups", () => {
    const rows = [row({ lane: 1, total: 5 }), row({ lane: 2, total: 9, withdrawn: true })];
    const groups = standingsGroups(rows, { groupByCategory: true, categoryOrder: order });
    expect(groups).toHaveLength(1);
    expect(groups[0].category).toBeNull();
    expect(groups[0].rows.map((r) => r.lane)).toEqual([1, 2]);
  });

  it("keeps a single category when it has class groups", () => {
    const rows = [row({ lane: 1, class_group: { value: 1, name: "Marksman" } })];
    const groups = standingsGroups(rows, { groupByCategory: true, categoryOrder: order });
    expect(groups).toHaveLength(1);
    expect(groups[0]).toMatchObject({ category: "Open", classGroup: "Marksman" });
  });

  it("returns a single empty group for no rows", () => {
    const groups = standingsGroups([], { groupByCategory: true, categoryOrder: order });
    expect(groups).toEqual([{ category: null, rows: [] }]);
  });
});
