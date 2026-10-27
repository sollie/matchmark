import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import ShooterPicker from "./ShooterPicker.svelte";
import { stubFetch } from "../test/fetch";

const doe = {
  id: 7,
  last_name: "Doe",
  first_name: "Jane",
  country: "SWE",
  club: "Alpha",
  waid: "W7",
};
const roe = { id: 8, last_name: "Roe", first_name: "Rick", country: "NOR", club: "Beta", waid: "" };

function setup(props: Record<string, unknown> = {}) {
  const calls = stubFetch(() => ({ body: { shooters: [doe, roe] } }));
  const onselect = vi.fn();
  render(ShooterPicker, { onselect, ...props });
  return { calls, onselect, user: userEvent.setup() };
}

describe("ShooterPicker", () => {
  it("searches server-side after a debounce and lists name, WAID, club and country", async () => {
    const { user, calls } = setup();
    await user.type(screen.getByRole("combobox", { name: "Shooter" }), "do");
    const option = await screen.findByRole("option", { name: /Jane Doe/ });
    expect(option.textContent).toContain("W7");
    expect(option.textContent).toContain("Alpha");
    expect(option.textContent).toContain("SWE");
    await waitFor(() => expect(calls.some((c) => c.search.includes("q=do"))).toBe(true));
    const search = calls.filter((c) => c.search.includes("q=do"));
    expect(search).toHaveLength(1);
    expect(search[0].search).toContain("limit=20");
  });

  it("selects with the keyboard", async () => {
    const { user, onselect } = setup();
    await user.type(screen.getByRole("combobox", { name: "Shooter" }), "o");
    await screen.findByRole("option", { name: /Jane Doe/ });
    await user.keyboard("{ArrowDown}{ArrowDown}{Enter}");
    expect(onselect).toHaveBeenCalledWith(expect.objectContaining({ id: 8 }));
    expect((screen.getByRole("combobox", { name: "Shooter" }) as HTMLInputElement).value).toBe(
      "Rick Roe",
    );
  });

  it("hides excluded shooters", async () => {
    const { user } = setup({ exclude: [7] });
    await user.type(screen.getByRole("combobox", { name: "Shooter" }), "o");
    await screen.findByRole("option", { name: /Rick Roe/ });
    expect(screen.queryByRole("option", { name: /Jane Doe/ })).toBeNull();
  });

  it("over-fetches by the number of excluded shooters", async () => {
    const { user, calls } = setup({ exclude: [7, 9, 10] });
    await user.type(screen.getByRole("combobox", { name: "Shooter" }), "o");
    await waitFor(() => expect(calls.some((c) => c.search.includes("limit=23"))).toBe(true));
  });

  it("shows a no-match state and offers the add action with the typed query", async () => {
    stubFetch(() => ({ body: { shooters: [] } }));
    const onadd = vi.fn();
    render(ShooterPicker, { onselect: vi.fn(), onadd });
    const user = userEvent.setup();
    await user.type(screen.getByRole("combobox", { name: "Shooter" }), "zzz");
    await waitFor(() => expect(screen.getByText("No match")).toBeTruthy());
    await user.click(screen.getByRole("option", { name: "Search WA1500 / add new" }));
    expect(onadd).toHaveBeenCalledWith("zzz");
  });
});
