import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { render, screen, within } from "@testing-library/svelte";
import Results from "./Results.svelte";
import { stubFetch } from "../test/fetch";
import { navigate } from "../lib/router.svelte";

const matches = [
  {
    id: 7,
    name: "Spring Match",
    date: "2026-05-01",
    course: "150",
    status: "finished",
    current_segment: 10,
  },
  {
    id: 8,
    name: "Live Match",
    date: "2026-06-01",
    course: "150",
    status: "active",
    current_segment: 2,
  },
];

let originalFetch: typeof fetch;
beforeEach(() => {
  originalFetch = globalThis.fetch;
});
afterEach(() => {
  globalThis.fetch = originalFetch;
  navigate("/");
});

describe("Results", () => {
  it("heads the segment columns with the rulebook segment names", async () => {
    navigate("/results/7");
    stubFetch((call) => {
      if (call.path === "/api/rulebook") {
        return {
          body: {
            courses: [
              {
                key: "150",
                label: "150 Course",
                categories: [],
                segments: [
                  { number: 1, name: "Match 1", stages: [1, 2], shots: 24 },
                  { number: 2, name: "Match 2", stages: [3], shots: 18 },
                ],
              },
            ],
          },
        };
      }
      if (!call.path.endsWith("/results")) return { body: { matches } };
      return {
        body: {
          match: {
            id: 7,
            name: "Spring Match",
            date: "2026-05-01",
            course: "150",
            course_label: "150 Course",
          },
          segments: [1, 2],
          categories: [
            {
              category: "Revolver 1500",
              did_not_finish: [],
              groups: [],
              rows: [
                {
                  rank: 1,
                  segments: [
                    { segment: 1, score: 110 },
                    { segment: 2, score: 120 },
                  ],
                  total: 230,
                },
              ],
            },
          ],
          new_shooters: [],
        },
      };
    });
    render(Results);
    expect(await screen.findByRole("columnheader", { name: "Match 1" })).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "Match 2" })).toBeTruthy();
    expect(screen.queryByRole("columnheader", { name: "S1" })).toBeNull();
  });

  it("shows a score card per shooter with ring counts per segment", async () => {
    navigate("/results/7");
    const counts = (x: number) => ({ x, "10": 1, "9": 2, "8": 3, "7": 4, "0": 5, miss: 6 });
    stubFetch((call) => {
      if (!call.path.endsWith("/results")) return { body: { matches } };
      return {
        body: {
          match: {
            id: 7,
            name: "Spring Match",
            date: "2026-05-01",
            course: "150",
            course_label: "150 Course",
          },
          segments: [1, 2],
          categories: [
            {
              category: "Revolver 1500",
              did_not_finish: [],
              groups: [],
              rows: [
                {
                  rank: 1,
                  last_name: "Berg",
                  first_name: "Ola",
                  segments: [{ segment: 1, score: 110, ring_counts: counts(7) }, { segment: 2 }],
                  total: 110,
                },
              ],
            },
          ],
          new_shooters: [],
        },
      };
    });
    render(Results);
    const card = await screen.findByRole("table", { name: "Score card for Ola Berg" });
    const rows = within(card).getAllByRole("row");
    expect(rows.map((r) => r.textContent)).toEqual([
      expect.stringContaining("Total"),
      "X77",
      "1011",
      "922",
      "833",
      "744",
      "055",
      "M66",
    ]);
  });

  it("lists withdrawn entries as DNF below the ranked rows", async () => {
    navigate("/results/7");
    stubFetch((call) => {
      if (!call.path.endsWith("/results")) return { body: { matches } };
      return {
        body: {
          match: {
            id: 7,
            name: "Spring Match",
            date: "2026-05-01",
            course: "150",
            course_label: "150 Course",
          },
          segments: [1],
          categories: [
            {
              category: "Revolver 1500",
              rows: [
                {
                  rank: 1,
                  last_name: "Berg",
                  first_name: "Ola",
                  segments: [{ segment: 1, score: 110 }],
                  total: 110,
                },
              ],
              groups: [],
              did_not_finish: [
                {
                  last_name: "Slow",
                  first_name: "Sam",
                  segments: [{ segment: 1, score: 80 }],
                },
              ],
            },
          ],
          new_shooters: [],
        },
      };
    });
    render(Results);
    const row = await screen.findByRole("row", { name: /Sam Slow/ });
    expect(row.textContent).toContain("80");
    expect(within(row).getByText("DNF")).toBeTruthy();
    const rows = screen.getAllByRole("row");
    expect(rows.indexOf(row)).toBeGreaterThan(
      rows.findIndex((r) => /Ola Berg/.test(r.textContent ?? "")),
    );
  });

  it("offers only finished matches", async () => {
    navigate("/results");
    stubFetch(() => ({ body: { matches } }));
    render(Results);
    expect(await screen.findByRole("option", { name: "Spring Match (2026-05-01)" })).toBeTruthy();
    expect(screen.queryByRole("option", { name: /Live Match/ })).toBeNull();
  });

  it("renders only the columns the server sent", async () => {
    navigate("/results/7");
    stubFetch((call) =>
      call.path.endsWith("/results")
        ? {
            body: {
              match: {
                id: 7,
                name: "Spring Match",
                date: "2026-05-01",
                course: "150",
                course_label: "150 Course",
              },
              segments: [1, 2],
              categories: [
                {
                  category: "Revolver 1500",
                  did_not_finish: [],
                  groups: [],
                  rows: [
                    {
                      rank: 1,
                      last_name: "Berg",
                      first_name: "Ola",
                      waid: "1001",
                      segments: [{ segment: 1 }, { segment: 2 }],
                      total: 230,
                    },
                  ],
                },
              ],
              new_shooters: [],
            },
          }
        : { body: { matches } },
    );
    render(Results);
    expect(await screen.findByRole("heading", { name: "Spring Match" })).toBeTruthy();
    expect(screen.getByRole("cell", { name: "Ola Berg" })).toBeTruthy();
    expect(screen.getByRole("cell", { name: "230" })).toBeTruthy();
    expect(screen.queryByRole("columnheader", { name: "S1" })).toBeNull();
    expect(screen.queryByRole("columnheader", { name: "X" })).toBeNull();
    expect(screen.getByRole("link", { name: "Download PDF" }).getAttribute("href")).toBe(
      "/api/matches/7/results?format=pdf",
    );
  });

  it("says so when the results are not available", async () => {
    navigate("/results/8");
    stubFetch((call) =>
      call.path.endsWith("/results")
        ? { status: 404, body: { error: "not found" } }
        : { body: { matches } },
    );
    render(Results);
    expect(await screen.findByText("Results are not available for this match.")).toBeTruthy();
  });

  it("shows a Classification column with the Classification name, hiding Unclassified", async () => {
    navigate("/results/7");
    stubFetch((call) => {
      if (!call.path.endsWith("/results")) return { body: { matches } };
      const row = (rank: number, last: string, classification?: object) => ({
        rank,
        last_name: last,
        first_name: "A",
        segments: [{ segment: 1, score: 100 }],
        total: 100,
        classification,
      });
      return {
        body: {
          match: {
            id: 7,
            name: "Spring Match",
            date: "2026-05-01",
            course: "150",
            course_label: "150",
          },
          segments: [1],
          categories: [
            {
              category: "Pistol 1500",
              did_not_finish: [],
              groups: [],
              rows: [
                row(1, "Berg", { value: 4, name: "Master" }),
                row(2, "Dahl", { value: 0, name: "Unclassified" }),
                row(3, "Eng"),
              ],
            },
          ],
          new_shooters: [],
        },
      };
    });
    render(Results);
    expect(await screen.findByRole("columnheader", { name: "Classification" })).toBeTruthy();
    const cells = (name: RegExp) =>
      within(screen.getByRole("row", { name }))
        .getAllByRole("cell")
        .map((c) => c.textContent);
    expect(cells(/Berg/)).toContain("Master");
    expect(cells(/Dahl/)).not.toContain("Unclassified");
    expect(cells(/Dahl/)).toHaveLength(cells(/Berg/).length);
  });

  it("omits the Classification column when no row has a Classification", async () => {
    navigate("/results/7");
    stubFetch((call) => {
      if (!call.path.endsWith("/results")) return { body: { matches } };
      return {
        body: {
          match: {
            id: 7,
            name: "Spring Match",
            date: "2026-05-01",
            course: "150",
            course_label: "150",
          },
          segments: [1],
          categories: [
            {
              category: "Pistol 1500",
              did_not_finish: [],
              groups: [],
              rows: [
                {
                  rank: 1,
                  last_name: "Berg",
                  first_name: "A",
                  classification: { value: 0, name: "Unclassified" },
                  segments: [{ segment: 1, score: 100 }],
                  total: 100,
                },
              ],
            },
          ],
          new_shooters: [],
        },
      };
    });
    render(Results);
    expect(await screen.findByRole("columnheader", { name: "Total" })).toBeTruthy();
    expect(screen.queryByRole("columnheader", { name: "Classification" })).toBeNull();
  });

  it("renders one ranked table per Class group under the category heading", async () => {
    navigate("/results/7");
    const row = (rank: number, total: number) => ({
      rank,
      segments: [{ segment: 1, score: total }],
      total,
    });
    stubFetch((call) => {
      if (!call.path.endsWith("/results")) return { body: { matches } };
      return {
        body: {
          match: {
            id: 7,
            name: "Spring Match",
            date: "2026-05-01",
            course: "150",
            course_label: "150",
          },
          segments: [1],
          categories: [
            {
              category: "Revolver 1500",
              did_not_finish: [],
              rows: [],
              groups: [
                { class: { value: 4, name: "Master" }, rows: [row(1, 230), row(2, 220)] },
                { class: { value: 1, name: "Marksman" }, rows: [row(1, 200)] },
              ],
            },
          ],
          new_shooters: [],
        },
      };
    });
    render(Results);
    await screen.findByRole("heading", { name: "Master" });
    expect(screen.getAllByRole("heading", { level: 4 }).map((h) => h.textContent)).toEqual([
      "Master",
      "Marksman",
    ]);
    const tables = screen.getAllByRole("table");
    expect(tables).toHaveLength(2);
    expect(within(tables[1]).getAllByRole("row")[1].textContent).toContain("1200");
  });
});
