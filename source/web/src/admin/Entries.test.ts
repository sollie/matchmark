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
    { stage_number: 1, expected_shots: 30, locked: true, scorer_lane: 2, scorer_name: "Rick Roe" },
    { stage_number: 2, expected_shots: 30, locked: false, scorer_lane: null, scorer_name: null },
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
  it("labels each stage as Locked or Open", () => {
    setup();
    const row = screen.getByRole("row", { name: /Doe, Jane/ });
    expect(within(row).getByText("Locked")).toBeTruthy();
    expect(within(row).getByText("Open")).toBeTruthy();
  });

  it("unlocks a locked stage and reloads", async () => {
    const { calls, reload, user } = setup();
    await user.click(screen.getByRole("button", { name: "Unlock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual(["POST /api/competitions/5/entries/1/stages/1/unlock"]);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("locks an open stage", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual(["POST /api/competitions/5/entries/1/stages/2/lock"]);
  });

  it("overwrites counts and locks the stage when asked", async () => {
    const { calls, user } = setup();
    await user.click(screen.getAllByRole("button", { name: "Edit" })[1]);
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

  it("saves counts without locking", async () => {
    const { calls, user } = setup();
    await user.click(screen.getAllByRole("button", { name: "Edit" })[1]);
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
