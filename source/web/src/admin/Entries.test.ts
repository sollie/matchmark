import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Entries from "./Entries.svelte";
import { stubFetch } from "../test/fetch";
import type { components } from "../lib/api/schema";

type EntryWithStages = components["schemas"]["EntryWithStages"];

const shooters = [
  { id: 7, last_name: "Doe", first_name: "Jane", sex: "F" as const, country: "SWE", club: "Club", waid: "" },
  { id: 8, last_name: "Roe", first_name: "Rick", sex: "M" as const, country: "SWE", club: "Club", waid: "" },
];

const entry = (overrides: Partial<EntryWithStages> = {}): EntryWithStages => ({
  id: 1,
  competition_id: 5,
  shooter_id: 7,
  category: "Pistol",
  lane: 1,
  withdrawn: false,
  stages: [
    { stage_number: 1, expected_shots: 30, status: "accepted", scorer_lane: 2, scorer_name: "Rick Roe", score: 287, shots_entered: 30 },
    { stage_number: 2, expected_shots: 30, status: "open", scorer_lane: null, scorer_name: null, score: 0, shots_entered: 0 },
  ],
  ...overrides,
});

function setup(entries: EntryWithStages[] = [entry()]) {
  const calls = stubFetch();
  const reload = vi.fn(async () => {});
  render(Entries, { competitionId: 5, entries, shooters, reload });
  return { calls, reload, user: userEvent.setup() };
}

afterEach(() => vi.restoreAllMocks());

describe("Entries", () => {
  it("labels each stage as Accepted or Open", () => {
    setup();
    const row = screen.getByRole("row", { name: /Jane Doe/ });
    expect(within(row).getByText("Accepted")).toBeTruthy();
    expect(within(row).getByText("Open")).toBeTruthy();
  });

  it("shows scores and tells scored-but-unlocked stages from untouched ones", () => {
    const stage = (n: number, status: "open" | "accepted", score: number, shots: number) => ({
      stage_number: n,
      expected_shots: 30,
      status,
      scorer_lane: null,
      scorer_name: null,
      score,
      shots_entered: shots,
    });
    setup([entry({ stages: [stage(1, "accepted", 287, 30), stage(2, "open", 295, 30), stage(3, "open", 0, 0)] })]);
    const row = screen.getByRole("row", { name: /Jane Doe/ });
    expect(within(row).getByText("Accepted")).toBeTruthy();
    expect(within(row).getByText("287")).toBeTruthy();
    expect(within(row).getByText("Scored, not locked")).toBeTruthy();
    expect(within(row).getByText("295")).toBeTruthy();
    expect(within(row).getAllByText("Open")).toHaveLength(1);
    expect(within(row).getAllByText("pts")).toHaveLength(2);
  });

  it("unlocks a locked stage and reloads", async () => {
    const { calls, reload, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 1: Unlock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual(["POST /api/competitions/5/entries/1/stages/1/unlock"]);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("locks an open stage", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 2: Lock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual(["POST /api/competitions/5/entries/1/stages/2/lock"]);
  });

  it("overwrites counts and locks the stage when asked", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 2: Edit" }));
    const x = screen.getByLabelText("X");
    await user.clear(x);
    await user.type(x, "30");
    await user.click(screen.getByRole("button", { name: "Save and lock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "PUT /api/competitions/5/entries/1/stages/2",
      "POST /api/competitions/5/entries/1/stages/2/lock",
    ]);
    expect(calls[0].body).toEqual({ x: 30, 10: 0, 9: 0, 8: 0, 7: 0, 0: 0, miss: 0 });
  });

  it("shows the entered total against the expected shots", async () => {
    const { user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 2: Edit" }));
    expect(screen.getByText("30 to go", { exact: false })).toBeTruthy();
    const x = screen.getByLabelText("X");
    await user.clear(x);
    await user.type(x, "30");
    expect(screen.getByText("Shots match", { exact: false })).toBeTruthy();
    await user.type(x, "1");
    expect(screen.getByText("271 too many", { exact: false })).toBeTruthy();
  });

  it("saves counts without locking", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 2: Edit" }));
    await user.click(screen.getByRole("button", { name: "Save counts" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual(["PUT /api/competitions/5/entries/1/stages/2"]);
  });

  it.each([
    ["Withdraw", "POST", "/api/competitions/5/entries/1/withdraw"],
    ["Remove", "DELETE", "/api/competitions/5/entries/1"],
  ])("asks before %s and only sends when confirmed", async (label, method, path) => {
    const { calls, user } = setup();
    const confirm = vi.spyOn(window, "confirm").mockReturnValueOnce(false).mockReturnValueOnce(true);
    await user.click(screen.getByRole("button", { name: label }));
    expect(calls).toHaveLength(0);
    await user.click(screen.getByRole("button", { name: label }));
    expect(confirm).toHaveBeenCalledTimes(2);
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([`${method} ${path}`]);
  });

  it("marks withdrawn entries in text", () => {
    setup([entry({ withdrawn: true })]);
    expect(screen.getByText("Withdrawn")).toBeTruthy();
  });
});

describe("Entries awaiting acceptance", () => {
  const awaiting = () =>
    entry({
      stages: [{ stage_number: 1, expected_shots: 30, status: "awaiting_acceptance", scorer_lane: 2, scorer_name: "Rick Roe", score: 287, shots_entered: 30 }],
    });

  it("accepts a stage that awaits acceptance", async () => {
    const { calls, reload, user } = setup([awaiting()]);
    expect(screen.getByText("Awaiting acceptance")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 1: Accept" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual(["POST /api/competitions/5/entries/1/stages/1/accept"]);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("offers no accept for an accepted stage", () => {
    setup();
    expect(screen.queryByRole("button", { name: "Lane 1 stage 1: Accept" })).toBeNull();
  });
});
