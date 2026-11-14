import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import CompetitionForm from "./CompetitionForm.svelte";
import { stubFetch } from "../test/fetch";

function setup() {
  const calls = stubFetch((call) => {
    if (call.path === "/api/ranges") return { body: { ranges: [] } };
    if (call.path === "/api/officials/candidates") return { body: { candidates: [] } };
    return undefined;
  });
  const saved = vi.fn();
  render(CompetitionForm, { saved, cancel: vi.fn() });
  return { calls, saved, user: userEvent.setup() };
}

const writes = (calls: ReturnType<typeof setup>["calls"]) =>
  calls.filter((c) => c.method !== "GET");

afterEach(() => vi.restoreAllMocks());

describe("CompetitionForm", () => {
  it("explains a missing name and saves nothing", async () => {
    const { calls, saved, user } = setup();
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect((await screen.findByRole("alert")).textContent).toBe("Enter a competition name.");
    expect(screen.getByLabelText("Competition name").getAttribute("aria-invalid")).toBe("true");
    expect(writes(calls)).toHaveLength(0);
    expect(saved).not.toHaveBeenCalled();
  });

  it("rejects an end date before the start date", async () => {
    const { calls, user } = setup();
    await user.type(screen.getByLabelText("Competition name"), "Spring");
    await user.clear(screen.getByLabelText("Start date"));
    await user.type(screen.getByLabelText("Start date"), "2026-03-02");
    await user.clear(screen.getByLabelText("End date"));
    await user.type(screen.getByLabelText("End date"), "2026-03-01");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect((await screen.findByRole("alert")).textContent).toBe(
      "End date must not be before the start date.",
    );
    expect(writes(calls)).toHaveLength(0);
  });

  it("saves a valid competition", async () => {
    const { calls, saved, user } = setup();
    await user.type(screen.getByLabelText("Competition name"), " Spring ");
    await user.click(screen.getByRole("button", { name: "Save" }));
    await vi.waitFor(() => expect(saved).toHaveBeenCalledOnce());
    expect(writes(calls)[0].body).toMatchObject({ name: "Spring" });
  });
});
