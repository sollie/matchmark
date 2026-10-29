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
              category: "Revolver",
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
              category: "Revolver",
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
                  category: "Revolver",
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
});
