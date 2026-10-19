import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Scorer from "./Scorer.svelte";
import { stubFetch } from "../test/fetch";

const session = (locked: boolean[]) => ({
  lane: 1,
  score_lane: 2,
  entry_id: 4,
  shooter_name: "Rick Roe",
  stages: locked.map((l, i) => ({ stage_number: i + 1, expected_shots: 30, locked: l })),
});

beforeEach(() => localStorage.setItem("mm_device_token", "device-token"));
afterEach(() => {
  vi.restoreAllMocks();
  localStorage.clear();
});

describe("Scorer", () => {
  it("shows the first open stage and saves counts for it", async () => {
    const calls = stubFetch((c) => (c.path === "/api/scorer/session" ? { body: session([true, false]) } : undefined));
    const user = userEvent.setup();
    render(Scorer);
    await screen.findByText("Scoring lane 2: Rick Roe");
    expect(screen.getByRole("tab", { name: /Stage 2/ }).getAttribute("aria-selected")).toBe("true");
    await user.click(screen.getByRole("button", { name: "Increase X" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    const save = calls.find((c) => c.method === "PUT");
    expect(save?.path).toBe("/api/scorer/stages/2");
    expect(save?.authorization).toBe("Bearer device-token");
    expect(save?.body).toMatchObject({ x: 1 });
  });

  it("shows a locked banner for a locked stage and saves nothing", async () => {
    const calls = stubFetch((c) => (c.path === "/api/scorer/session" ? { body: session([true, false]) } : undefined));
    const user = userEvent.setup();
    render(Scorer);
    await screen.findByText("Scoring lane 2: Rick Roe");
    await user.click(screen.getByRole("tab", { name: /Stage 1/ }));
    expect(screen.getByText(/Stage 1 is locked/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save" })).toBeNull();
    expect(calls.some((c) => c.method === "PUT")).toBe(false);
  });
});
