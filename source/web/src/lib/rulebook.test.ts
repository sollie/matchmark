import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { stubFetch } from "../test/fetch";

const body = {
  courses: [{ key: "60", label: "60 shots", discipline: "60", total_shots: 60, stages: [] }],
  rings: [],
};

beforeEach(() => vi.resetModules());
afterEach(() => vi.restoreAllMocks());

describe("rulebook store", () => {
  it("loads the courses once and resolves labels", async () => {
    const calls = stubFetch(() => ({ body }));
    const { loadRulebook, rulebook, courseLabel } = await import("./rulebook.svelte");
    await loadRulebook();
    await loadRulebook();
    expect(calls.map((c) => c.path)).toEqual(["/api/rulebook"]);
    expect(rulebook.courses).toHaveLength(1);
    expect(courseLabel("60")).toBe("60 shots");
  });

  it("keeps the key as the label and retries after a failure", async () => {
    let fail = true;
    const calls = stubFetch(() => (fail ? { status: 500, body: { error: "boom" } } : { body }));
    const { loadRulebook, rulebook, courseLabel } = await import("./rulebook.svelte");
    await loadRulebook();
    expect(rulebook.courses).toHaveLength(0);
    expect(courseLabel("60")).toBe("60");
    fail = false;
    await loadRulebook();
    expect(calls).toHaveLength(2);
    expect(courseLabel("60")).toBe("60 shots");
  });

  it("labels stages with their Part, or the plain number when the course has none", async () => {
    const stage = (n: number, part?: string) => ({
      stage_number: n,
      distance: "7 yd",
      expected_shots: 12,
      time_seconds: 20,
      position: "",
      part,
    });
    const grouped = {
      courses: [
        {
          key: "150",
          label: "150",
          discipline: "1500",
          total_shots: 150,
          categories: [],
          stages: [stage(4, "Match 3")],
        },
        {
          key: "60",
          label: "60",
          discipline: "60",
          total_shots: 60,
          categories: [],
          stages: [stage(4)],
        },
      ],
      rings: [],
    };
    stubFetch(() => ({ body: grouped }));
    const { loadRulebook, stageLabel } = await import("./rulebook.svelte");
    await loadRulebook();
    expect(stageLabel("150", 4)).toBe("Stage 4 · Match 3");
    expect(stageLabel("60", 4)).toBe("Stage 4");
    expect(stageLabel("unknown", 4)).toBe("Stage 4");
  });
});
