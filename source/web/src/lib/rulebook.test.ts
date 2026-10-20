import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { stubFetch } from "../test/fetch";

const body = {
  courses: [{ key: "60", label: "60 shots", discipline: "60", total_shots: 60, stages: [] }],
  weapon_classes: {},
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
});
