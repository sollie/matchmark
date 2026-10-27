import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Entries from "./Entries.svelte";
import { stubFetch } from "../test/fetch";
import { rulebook, type RulebookCourse } from "../lib/rulebook.svelte";
import type { components } from "../lib/api/schema";

type EntryWithStages = components["schemas"]["EntryWithStages"];

const shooters = [
  { id: 7, last_name: "Doe", first_name: "Jane", country: "SWE", club: "Club", waid: "" },
  { id: 8, last_name: "Roe", first_name: "Rick", country: "SWE", club: "Club", waid: "" },
];

const entry = (overrides: Partial<EntryWithStages> = {}): EntryWithStages => ({
  id: 1,
  match_id: 5,
  shooter_id: 7,
  category: "Pistol",
  lane: 1,
  withdrawn: false,
  stages: [
    {
      stage_number: 1,
      expected_shots: 30,
      status: "accepted",
      scorer_lane: 2,
      scorer_name: "Rick Roe",
      score: 287,
      shots_entered: 30,
    },
    {
      stage_number: 2,
      expected_shots: 30,
      status: "open",
      scorer_lane: null,
      scorer_name: null,
      score: 0,
      shots_entered: 0,
    },
  ],
  ...overrides,
});

const course = (key: string, categories: string[]) =>
  ({
    key,
    label: key,
    categories,
    stages: key === "150" ? [{ stage_number: 1, part: "Match 1" }] : [],
  }) as unknown as RulebookCourse;

function setup(entries: EntryWithStages[] = [entry()], courseKey = "150") {
  rulebook.courses = [
    course("150", ["Revolver", "Pistol", "Revolver OS", "Pistol OS"]),
    course("40", ["Revolver"]),
  ];
  const calls = stubFetch();
  const reload = vi.fn(async () => {});
  const view = render(Entries, { matchId: 5, course: courseKey, entries, shooters, reload });
  return { calls, reload, view, user: userEvent.setup() };
}

const options = (name: string) =>
  within(screen.getByRole("combobox", { name }))
    .getAllByRole("option")
    .map((o) => o.textContent);

afterEach(() => vi.restoreAllMocks());

describe("Entries", () => {
  it("offers only the categories the course allows when adding and editing", async () => {
    const { user } = setup([entry({ category: "Revolver" })], "40");
    expect(options("Entry category")).toEqual(["Revolver"]);
    await user.click(screen.getByRole("button", { name: "Move" }));
    expect(options("Move category")).toEqual(["Revolver"]);
  });

  it("titles stage columns with the Part for grouped courses only", () => {
    setup([entry()], "150");
    expect(screen.getByRole("columnheader", { name: "S1" }).getAttribute("title")).toBe(
      "Stage 1 · Match 1",
    );
    expect(screen.getByRole("columnheader", { name: "S2" }).getAttribute("title")).toBe("Stage 2");
  });

  it("keeps the current category of a legacy entry selectable when editing", async () => {
    const { user } = setup([entry({ category: "Pistol" })], "40");
    await user.click(screen.getByRole("button", { name: "Move" }));
    expect(options("Move category")).toEqual(["Pistol", "Revolver"]);
  });

  it("falls back to the first allowed category when the course changes", async () => {
    const { view, user } = setup([], "150");
    await user.selectOptions(screen.getByRole("combobox", { name: "Entry category" }), "Pistol OS");
    await view.rerender({ course: "40" });
    expect(options("Entry category")).toEqual(["Revolver"]);
    expect(
      (screen.getByRole("combobox", { name: "Entry category" }) as HTMLSelectElement).value,
    ).toBe("Revolver");
  });

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
    setup([
      entry({
        stages: [stage(1, "accepted", 287, 30), stage(2, "open", 295, 30), stage(3, "open", 0, 0)],
      }),
    ]);
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
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "POST /api/matches/5/entries/1/stages/1/unlock",
    ]);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("locks an open stage", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 2: Lock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "POST /api/matches/5/entries/1/stages/2/lock",
    ]);
  });

  it("overwrites counts and locks the stage when asked", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 2: Edit" }));
    const x = screen.getByLabelText("X");
    await user.clear(x);
    await user.type(x, "30");
    await user.click(screen.getByRole("button", { name: "Save and lock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "PUT /api/matches/5/entries/1/stages/2",
      "POST /api/matches/5/entries/1/stages/2/lock",
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
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "PUT /api/matches/5/entries/1/stages/2",
    ]);
  });

  it.each([
    ["Withdraw", "POST", "/api/matches/5/entries/1/withdraw"],
    ["Remove", "DELETE", "/api/matches/5/entries/1"],
  ])("asks before %s and only sends when confirmed", async (label, method, path) => {
    const { calls, user } = setup();
    const confirm = vi
      .spyOn(window, "confirm")
      .mockReturnValueOnce(false)
      .mockReturnValueOnce(true);
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
      stages: [
        {
          stage_number: 1,
          expected_shots: 30,
          status: "awaiting_acceptance",
          scorer_lane: 2,
          scorer_name: "Rick Roe",
          score: 287,
          shots_entered: 30,
        },
      ],
    });

  it("accepts a stage that awaits acceptance", async () => {
    const { calls, reload, user } = setup([awaiting()]);
    expect(screen.getByText("Awaiting acceptance")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Lane 1 stage 1: Accept" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "POST /api/matches/5/entries/1/stages/1/accept",
    ]);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("offers no accept for an accepted stage", () => {
    setup();
    expect(screen.queryByRole("button", { name: "Lane 1 stage 1: Accept" })).toBeNull();
  });

  it("imports a shooter from WA1500 inside the picker and selects it for the entry", async () => {
    const imported = {
      id: 11,
      last_name: "Anderson",
      first_name: "Mike",
      country: "SWE",
      club: "Alpha",
      waid: "W100",
    };
    const calls = stubFetch((c) => {
      if (c.path === "/api/shooters" && c.method === "GET") return { body: { shooters: [] } };
      if (c.path === "/api/wa1500/search")
        return {
          body: {
            results: [
              {
                waid: "W100",
                first_name: "Mike",
                last_name: "Anderson",
                country: "SWE",
                club: "Alpha",
              },
            ],
          },
        };
      if (c.path === "/api/shooters/import")
        return { status: 201, body: { id: 11, action: "created" } };
      if (c.path === "/api/shooters/11") return { body: imported };
      return { status: 201, body: {} };
    });
    render(Entries, {
      matchId: 5,
      course: "150",
      entries: [],
      shooters,
      reload: vi.fn(async () => {}),
    });
    const user = userEvent.setup();
    await user.type(screen.getByRole("combobox", { name: "Shooter" }), "ande");
    await user.click(await screen.findByRole("option", { name: "Search WA1500 / add new" }));
    expect((screen.getByRole("textbox", { name: "Search WA1500" }) as HTMLInputElement).value).toBe(
      "ande",
    );
    await user.click(screen.getByRole("button", { name: "Search" }));
    await user.click(screen.getByRole("button", { name: "Import Mike Anderson" }));
    await waitFor(() =>
      expect((screen.getByRole("combobox", { name: "Shooter" }) as HTMLInputElement).value).toBe(
        "Mike Anderson",
      ),
    );
    expect(screen.queryByRole("textbox", { name: "Search WA1500" })).toBeNull();
    await user.click(screen.getByRole("button", { name: "Add entry" }));
    const post = calls.find((c) => c.method === "POST" && c.path === "/api/matches/5/entries");
    expect(post?.body).toMatchObject({ shooter_id: 11 });
  });
});
