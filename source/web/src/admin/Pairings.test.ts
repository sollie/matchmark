import { describe, expect, it } from "vitest";
import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Pairings from "./Pairings.svelte";
import { stubFetch } from "../test/fetch";

const entries = [
  { id: 1, lane: 1, withdrawn: false },
  { id: 2, lane: 2, withdrawn: false },
] as never[];

function setup() {
  const calls = stubFetch((call) => {
    if (call.path === "/api/matches/5/pairings" && call.method === "GET")
      return {
        body: {
          pairings: [
            { id: 1, lane: 1, revoked: false },
            { id: 2, lane: 2, revoked: true },
          ],
        },
      };
    return { body: {} };
  });
  render(Pairings, { matchId: 5, entries, finished: false });
  return { calls, user: userEvent.setup() };
}

describe("Pairings", () => {
  it("shows which devices are active and revokes one", async () => {
    const { calls, user } = setup();
    expect(await screen.findByText("1 active device")).toBeTruthy();
    expect(screen.getAllByText("paired")).toHaveLength(1);
    expect(screen.getAllByText("revoked")).toHaveLength(1);
    await user.click(screen.getByRole("button", { name: "Revoke" }));
    await waitFor(() =>
      expect(
        calls.some((c) => c.method === "DELETE" && c.path === "/api/matches/5/pairings/1"),
      ).toBe(true),
    );
  });
});
