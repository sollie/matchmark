import { describe, expect, it } from "vitest";
import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Screens from "./Screens.svelte";
import { stubFetch, type Call } from "../test/fetch";

const screenRow = {
  id: 3,
  name: "Gate",
  match_id: null,
  online: true,
  last_seen_at: "2026-01-01T00:00:00Z",
};
const matches = [{ id: 7, name: "Club match", date: "2026-01-01", course: "40", status: "draft" }];

function setup(respond?: (call: Call) => { status?: number; body?: unknown } | undefined) {
  const calls = stubFetch((call) => {
    const custom = respond?.(call);
    if (custom) return custom;
    if (call.path === "/api/screens" && call.method === "GET")
      return { body: { screens: [screenRow] } };
    if (call.path === "/api/screens/assignable-matches") return { body: { matches } };
    if (call.path === "/api/screens/stream") return { status: 401 };
    return { body: screenRow };
  });
  render(Screens);
  return { calls, user: userEvent.setup() };
}

describe("Screens", () => {
  it("lists screens with online state and assigns a match", async () => {
    const { calls, user } = setup();
    expect(await screen.findByText("Gate")).toBeTruthy();
    expect(screen.getByText("Online")).toBeTruthy();
    await user.selectOptions(screen.getByLabelText("Match for Gate"), "7");
    await waitFor(() =>
      expect(calls.find((c) => c.method === "PUT" && c.path === "/api/screens/3")?.body).toEqual({
        match_id: 7,
      }),
    );
  });

  it("approves a pending code", async () => {
    const { calls, user } = setup();
    await screen.findByText("Gate");
    await user.type(screen.getByLabelText("Pending code"), "ABC234");
    await user.click(screen.getByRole("button", { name: "Approve screen" }));
    await waitFor(() =>
      expect(calls.find((c) => c.path === "/api/screens/approve")?.body).toEqual({
        code: "ABC234",
        name: "",
      }),
    );
  });

  it("applies a match to all screens and removes a screen", async () => {
    const { calls, user } = setup();
    await screen.findByText("Gate");
    await user.selectOptions(screen.getByLabelText("Match for all screens"), "7");
    await user.click(screen.getByRole("button", { name: "Apply to all screens" }));
    await waitFor(() =>
      expect(calls.find((c) => c.path === "/api/screens/assignment")?.body).toEqual({
        match_id: 7,
      }),
    );
    await user.click(screen.getByRole("button", { name: "Remove" }));
    await waitFor(() =>
      expect(calls.some((c) => c.method === "DELETE" && c.path === "/api/screens/3")).toBe(true),
    );
  });
});
