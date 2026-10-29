import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import MatchEdit from "./MatchEdit.svelte";
import { stubFetch } from "../test/fetch";
import { loadRulebook } from "../lib/rulebook.svelte";

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
};

function setup(hasEntries = false, reply?: Parameters<typeof stubFetch>[0]) {
  const calls = stubFetch(reply);
  const saved = vi.fn();
  const cancel = vi.fn();
  render(MatchEdit, { match, hasEntries, saved, cancel });
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
    await user.selectOptions(screen.getByLabelText("Course of fire"), "60");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(calls).toHaveLength(1);
    expect(calls[0].method).toBe("PUT");
    expect(calls[0].path).toBe("/api/matches/5");
    expect(calls[0].body).toEqual({ name: "Autumn", date: "2026-10-19", course: "60" });
    expect(saved).toHaveBeenCalledOnce();
  });

  it("disables the course and explains why once entries exist", () => {
    setup(true);
    expect((screen.getByLabelText("Course of fire") as HTMLSelectElement).disabled).toBe(true);
    expect(screen.getByText(/cannot change once the match has entries/)).toBeTruthy();
  });

  it("blocks a date that is not ISO 8601", async () => {
    const { calls, user } = setup();
    await user.clear(screen.getByLabelText("Date"));
    await user.type(screen.getByLabelText("Date"), "19/10/2026");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(calls).toHaveLength(0);
    expect((screen.getByLabelText("Date") as HTMLInputElement).validity.valid).toBe(false);
  });

  it("shows API errors inline", async () => {
    const { saved, user } = setup(false, () => ({ status: 409, body: { error: "conflict" } }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect((await screen.findByRole("alert")).textContent).toBe("conflict");
    expect(saved).not.toHaveBeenCalled();
  });

  it("cancels without calling the API", async () => {
    const { calls, cancel, user } = setup();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(cancel).toHaveBeenCalledOnce();
    expect(calls).toHaveLength(0);
  });
});
