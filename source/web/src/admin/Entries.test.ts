import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Entries from "./Entries.svelte";
import { stubFetch } from "../test/fetch";
import { rulebook, type RulebookCourse } from "../lib/rulebook.svelte";
import type { components } from "../lib/api/schema";

type EntryWithSegments = components["schemas"]["EntryWithSegments"];

const shooters = [
  {
    id: 7,
    last_name: "Doe",
    first_name: "Jane",
    country: "SWE",
    club: "Club",
    waid: "",
    archived: false,
  },
  {
    id: 8,
    last_name: "Roe",
    first_name: "Rick",
    country: "SWE",
    club: "Club",
    waid: "",
    archived: false,
  },
];

const entry = (overrides: Partial<EntryWithSegments> = {}): EntryWithSegments => ({
  id: 1,
  match_id: 5,
  shooter_id: 7,
  category: "Pistol",
  lane: 1,
  withdrawn: false,
  segments: [
    {
      segment_number: 1,
      expected_shots: 30,
      status: "accepted",
      scorer_lane: 2,
      scorer_name: "Rick Roe",
      score: 287,
      shots_entered: 30,
      x: 0,
      "10": 0,
      "9": 0,
      "8": 0,
      "7": 0,
      "0": 0,
      miss: 0,
    },
    {
      segment_number: 2,
      expected_shots: 30,
      status: "open",
      scorer_lane: null,
      scorer_name: null,
      score: 0,
      shots_entered: 0,
      x: 0,
      "10": 0,
      "9": 0,
      "8": 0,
      "7": 0,
      "0": 0,
      miss: 0,
    },
  ],
  ...overrides,
});

const course = (key: string, categories: string[]) =>
  ({
    key,
    label: key,
    categories,
    segments: key === "150" ? [{ number: 1, name: "Match 1", stages: [1, 2], shots: 24 }] : [],
  }) as unknown as RulebookCourse;

function setup(
  entries: EntryWithSegments[] = [entry()],
  courseKey = "150",
  respond?: Parameters<typeof stubFetch>[0],
) {
  rulebook.courses = [
    course("150", ["Revolver", "Pistol", "Revolver OS", "Pistol OS"]),
    course("40", ["Revolver"]),
  ];
  const calls = stubFetch(respond);
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

  it("titles segment columns with the rulebook segment name", () => {
    setup([entry()], "150");
    expect(screen.getByRole("columnheader", { name: "M1" }).getAttribute("title")).toBe("Match 1");
    expect(screen.getByRole("columnheader", { name: "Segment 2" }).getAttribute("title")).toBe(
      "Segment 2",
    );
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

  it("labels each segment as Accepted or Open", () => {
    setup();
    const row = screen.getByRole("row", { name: /Jane Doe/ });
    expect(within(row).getByText("Accepted")).toBeTruthy();
    expect(within(row).getByText("Open")).toBeTruthy();
  });

  it("shows scores and tells scored-but-unlocked segments from untouched ones", () => {
    const segment = (n: number, status: "open" | "accepted", score: number, shots: number) => ({
      segment_number: n,
      expected_shots: 30,
      status,
      scorer_lane: null,
      scorer_name: null,
      score,
      shots_entered: shots,
      x: 0,
      "10": 0,
      "9": 0,
      "8": 0,
      "7": 0,
      "0": 0,
      miss: 0,
    });
    setup([
      entry({
        segments: [
          segment(1, "accepted", 287, 30),
          segment(2, "open", 295, 30),
          segment(3, "open", 0, 0),
        ],
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

  it("unlocks a locked segment and reloads", async () => {
    const { calls, reload, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 1: Unlock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "POST /api/matches/5/entries/1/segments/1/unlock",
    ]);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("locks an open segment", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 2: Lock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "POST /api/matches/5/entries/1/segments/2/lock",
    ]);
  });

  it("overwrites counts and locks the segment when asked", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 2: Edit" }));
    const x = screen.getByLabelText("X");
    await user.clear(x);
    await user.type(x, "30");
    await user.click(screen.getByRole("button", { name: "Save and lock" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "PUT /api/matches/5/entries/1/segments/2",
      "POST /api/matches/5/entries/1/segments/2/lock",
    ]);
    expect(calls[0].body).toEqual({ x: 30, 10: 0, 9: 0, 8: 0, 7: 0, 0: 0, miss: 0 });
  });

  it("shows the entered total against the expected shots", async () => {
    const { user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 2: Edit" }));
    expect(screen.getByText("30 to go", { exact: false })).toBeTruthy();
    const x = screen.getByLabelText("X");
    await user.clear(x);
    await user.type(x, "30");
    expect(screen.getByText("Shots match", { exact: false })).toBeTruthy();
    await user.type(x, "1");
    expect(screen.getByText("271 too many", { exact: false })).toBeTruthy();
  });

  it("edits counts in a modal dialog named after the shooter and segment, focusing the first input", async () => {
    const { user } = setup();
    const edit = screen.getByRole("button", { name: "Lane 1 segment 2: Edit" });
    await user.click(edit);
    const dialog = screen.getByRole("dialog", { name: "Jane Doe, Segment 2" });
    expect(dialog.hasAttribute("open")).toBe(true);
    expect(document.activeElement).toBe(within(dialog).getByLabelText("X"));
    await user.click(within(dialog).getByRole("button", { name: "Close" }));
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(edit);
  });

  it("does not submit when Enter is pressed in an input", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 2: Edit" }));
    await user.type(screen.getByLabelText("X"), "3{Enter}");
    expect(calls).toEqual([]);
    expect(screen.getByRole("dialog", { name: "Jane Doe, Segment 2" })).toBeTruthy();
  });

  it("closes on Cancel", async () => {
    const { user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 2: Edit" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("closes after a successful save", async () => {
    const { user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 2: Edit" }));
    await user.click(screen.getByRole("button", { name: "Save counts" }));
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("stays open with inputs kept and the error inside the dialog when saving fails", async () => {
    const { user } = setup([entry()], "150", () => ({
      status: 422,
      body: { error: "invalid input" },
    }));
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 2: Edit" }));
    const x = screen.getByLabelText("X");
    await user.type(x, "5");
    await user.click(screen.getByRole("button", { name: "Save and lock" }));
    const dialog = screen.getByRole("dialog", { name: "Jane Doe, Segment 2" });
    expect(within(dialog).getByRole("alert").textContent).toBe("invalid input");
    expect((within(dialog).getByLabelText("X") as HTMLInputElement).value).toBe("5");
  });

  it("saves counts without locking", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 2: Edit" }));
    await user.click(screen.getByRole("button", { name: "Save counts" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "PUT /api/matches/5/entries/1/segments/2",
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
      segments: [
        {
          segment_number: 1,
          expected_shots: 30,
          status: "awaiting_acceptance",
          scorer_lane: 2,
          scorer_name: "Rick Roe",
          score: 287,
          shots_entered: 30,
          x: 0,
          "10": 0,
          "9": 0,
          "8": 0,
          "7": 0,
          "0": 0,
          miss: 0,
        },
      ],
    });

  it("accepts a segment that awaits acceptance", async () => {
    const { calls, reload, user } = setup([awaiting()]);
    expect(screen.getByText("Awaiting acceptance")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Lane 1 segment 1: Accept" }));
    expect(calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "POST /api/matches/5/entries/1/segments/1/accept",
    ]);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("offers no accept for an accepted segment", () => {
    setup();
    expect(screen.queryByRole("button", { name: "Lane 1 segment 1: Accept" })).toBeNull();
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

  it("explains an archived shooter and offers an inline unarchive", async () => {
    const calls = stubFetch((c) => {
      if (c.path === "/api/shooters" && c.method === "GET")
        return { body: { shooters: [shooters[0]] } };
      if (c.path === "/api/matches/5/entries")
        return { status: 409, body: { error: "shooter_archived" } };
      return { body: { ...shooters[0] } };
    });
    render(Entries, {
      matchId: 5,
      course: "150",
      entries: [],
      shooters,
      reload: vi.fn(async () => {}),
    });
    const user = userEvent.setup();
    await user.type(screen.getByRole("combobox", { name: "Shooter" }), "Doe");
    await user.click(await screen.findByRole("option", { name: /Jane Doe/ }));
    await user.click(screen.getByRole("button", { name: "Add entry" }));
    expect(await screen.findByText(/This shooter is archived/)).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Unarchive" }));
    await waitFor(() => expect(screen.queryByText(/This shooter is archived/)).toBeNull());
    expect(calls.some((c) => c.method === "POST" && c.path === "/api/shooters/7/unarchive")).toBe(
      true,
    );
  });
});
