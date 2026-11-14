import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import MatchEdit from "./MatchEdit.svelte";
import { stubFetch, type Call } from "../test/fetch";
import { loadRulebook } from "../lib/rulebook.svelte";
import { choose } from "../test/select";

const course = (key: string, label: string) => ({
  key,
  label,
  discipline: "1500",
  total_shots: 0,
  segments: [],
});

beforeAll(async () => {
  stubFetch(() => ({
    body: {
      courses: [course("150", "150 shots (WA1500)"), course("60", "60 shots")],
      rings: [],
    },
  }));
  await loadRulebook();
});

const match = {
  id: 5,
  competition_id: 1,
  name: "Spring",
  date: "2026-03-01",
  course: "150",
  status: "draft" as const,
  current_segment: 1,
  classified: true,
  referees: [{ waid: "1001", name: "Rita Referee" }],
  range_officers: [],
  effective_referees: [{ waid: "1001", name: "Rita Referee" }],
  effective_range_officers: [],
};

const candidates = [
  { username: "rita", waid: "1001" },
  { username: "ola", waid: "2002" },
];

function saves(calls: Call[]) {
  return calls.filter((c) => c.method !== "GET");
}

function setup(hasEntries = false, reply?: Parameters<typeof stubFetch>[0], draft = true) {
  const calls = stubFetch((call) =>
    call.path === "/api/officials/candidates" ? { body: { candidates } } : reply?.(call),
  );
  const saved = vi.fn();
  const cancel = vi.fn();
  render(MatchEdit, { match, hasEntries, draft, saved, cancel });
  return { calls, saved, cancel, user: userEvent.setup() };
}

afterEach(() => vi.restoreAllMocks());

describe("MatchEdit", () => {
  it("submits the edited name, date and course", async () => {
    const { calls, saved, user } = setup(false, () => ({
      body: { ...match, name: "Autumn", date: "2026-10-19", course: "60" },
    }));
    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Autumn");
    await user.clear(screen.getByLabelText("Date"));
    await user.type(screen.getByLabelText("Date"), "2026-10-19");
    await choose(user, "Course of fire", "60 shots");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(saves(calls)).toHaveLength(1);
    expect(saves(calls)[0].method).toBe("PUT");
    expect(saves(calls)[0].path).toBe("/api/matches/5");
    expect(saves(calls)[0].body).toEqual({
      name: "Autumn",
      date: "2026-10-19",
      course: "60",
      classified: true,
      referee_waids: ["1001"],
      range_officer_waids: [],
    });
    expect(saved).toHaveBeenCalledOnce();
  });

  it("disables the course and explains why once entries exist", () => {
    setup(true);
    expect(
      (screen.getByRole("combobox", { name: "Course of fire" }) as HTMLButtonElement).disabled,
    ).toBe(true);
    expect(screen.getByText(/cannot change once the match has entries/)).toBeTruthy();
  });

  it("leaves only the name editable once the match has started", async () => {
    const { calls, user } = setup(false, () => ({ body: { ...match, name: "Renamed" } }), false);
    expect((screen.getByLabelText("Date") as HTMLInputElement).disabled).toBe(true);
    expect(
      (screen.getByRole("combobox", { name: "Course of fire" }) as HTMLButtonElement).disabled,
    ).toBe(true);
    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Renamed");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(saves(calls)[0].body).toEqual({
      name: "Renamed",
      date: match.date,
      course: match.course,
      classified: true,
      referee_waids: ["1001"],
      range_officer_waids: [],
    });
  });

  it("adds, removes and orders officials", async () => {
    const { calls, user } = setup();
    await user.click(screen.getByRole("button", { name: "Add referee" }));
    await user.type(screen.getByLabelText("Referee WAID 2"), " 2002 ");
    await user.click(screen.getByRole("button", { name: "Add range officer" }));
    await user.type(screen.getByLabelText("Range officer WAID 1"), "3003");
    await user.click(screen.getByRole("button", { name: "Add range officer" }));
    await user.type(screen.getByLabelText("Range officer WAID 2"), "4004");
    await user.click(screen.getByRole("button", { name: "Remove range officer 1" }));
    await user.click(screen.getByRole("button", { name: "Remove referee 1" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(saves(calls)[0].body).toMatchObject({
      referee_waids: ["2002"],
      range_officer_waids: ["4004"],
    });
  });

  it("picks officials from users that have a WAID, leaving out those already listed", async () => {
    const { calls, user } = setup();
    await screen.findByRole("combobox", { name: "Pick range officer from users" });
    await user.click(screen.getByRole("combobox", { name: "Pick referee from users" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent?.trim())).toEqual([
      "ola (2002)",
    ]);
    await user.keyboard("{Escape}");
    await choose(user, "Pick range officer from users", "ola (2002)");
    expect((screen.getByLabelText("Range officer WAID 1") as HTMLInputElement).value).toBe("2002");
    await user.click(screen.getByRole("combobox", { name: "Pick range officer from users" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent?.trim())).toEqual([
      "rita (1001)",
    ]);
    await user.keyboard("{Escape}");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(saves(calls)[0].body).toMatchObject({ range_officer_waids: ["2002"] });
  });

  it("sends the Classified match toggle, also once the match has started", async () => {
    const { calls, user } = setup(false, () => ({ body: { ...match, classified: false } }), false);
    await user.click(screen.getByLabelText("Classified match"));
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(saves(calls)[0].body).toMatchObject({ classified: false });
  });

  it("blocks a date that is not ISO 8601", async () => {
    const { calls, user } = setup();
    await user.clear(screen.getByLabelText("Date"));
    await user.type(screen.getByLabelText("Date"), "19/10/2026");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(saves(calls)).toHaveLength(0);
    expect(screen.getByRole("alert").textContent).toBe("Enter a date as YYYY-MM-DD.");
    expect(screen.getByLabelText("Date").getAttribute("aria-invalid")).toBe("true");
  });

  it("shows API errors inline", async () => {
    const { saved, user } = setup(false, () => ({ status: 409, body: { error: "conflict" } }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect((await screen.findByRole("alert")).textContent?.trim()).toBe("conflict");
    expect(saved).not.toHaveBeenCalled();
  });

  it("cancels without calling the API", async () => {
    const { calls, cancel, user } = setup();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(cancel).toHaveBeenCalledOnce();
    expect(saves(calls)).toHaveLength(0);
  });
});
