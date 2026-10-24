import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import Results from "./Results.svelte";
import { stubFetch } from "../test/fetch";
import { navigate } from "../lib/router.svelte";

const competitions = [
  { id: 7, name: "Spring Match", date: "2026-05-01", course: "150", status: "finished", current_stage: 10 },
  { id: 8, name: "Live Match", date: "2026-06-01", course: "150", status: "active", current_stage: 2 },
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
  it("offers only finished competitions", async () => {
    navigate("/results");
    stubFetch(() => ({ body: { competitions } }));
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
              competition: { id: 7, name: "Spring Match", date: "2026-05-01", course: "150", course_label: "150 Course" },
              stages: [1, 2],
              categories: [
                {
                  category: "Revolver",
                  rows: [{ rank: 1, last_name: "Berg", first_name: "Ola", waid: "1001", stages: [{ stage: 1 }, { stage: 2 }], total: 230 }],
                },
              ],
              new_shooters: [],
            },
          }
        : { body: { competitions } },
    );
    render(Results);
    expect(await screen.findByRole("heading", { name: "Spring Match" })).toBeTruthy();
    expect(screen.getByRole("cell", { name: "Ola Berg" })).toBeTruthy();
    expect(screen.getByRole("cell", { name: "230" })).toBeTruthy();
    expect(screen.queryByRole("columnheader", { name: "S1" })).toBeNull();
    expect(screen.queryByRole("columnheader", { name: "X" })).toBeNull();
  });

  it("says so when the results are not available", async () => {
    navigate("/results/8");
    stubFetch((call) => (call.path.endsWith("/results") ? { status: 404, body: { error: "not found" } } : { body: { competitions } }));
    render(Results);
    expect(await screen.findByText("Results are not available for this competition.")).toBeTruthy();
  });
});
