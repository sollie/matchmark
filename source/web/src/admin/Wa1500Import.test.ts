import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Wa1500Import from "./Wa1500Import.svelte";
import { stubFetch, type Call } from "../test/fetch";

const hit = { waid: "W100", first_name: "Mike", last_name: "Anderson", country: "SWE", club: "Alpha" };
const saved = { id: 5, last_name: "Anderson", first_name: "Mike", country: "SWE", club: "Alpha", waid: "W100" };
const other = { ...saved, id: 9, waid: "", club: "Beta" };

function setup(respond: (call: Call) => { status?: number; body?: unknown } | undefined) {
  const calls = stubFetch((call) => {
    if (call.path === "/api/shooters/5" || call.path === "/api/shooters/9") return { body: call.path.endsWith("9") ? other : saved };
    return respond(call);
  });
  const onsaved = vi.fn();
  render(Wa1500Import, { onsaved });
  return { calls, onsaved, user: userEvent.setup() };
}

async function search(user: ReturnType<typeof userEvent.setup>, text = "anderson") {
  await user.type(screen.getByRole("textbox", { name: "Search WA1500" }), text);
  await user.click(screen.getByRole("button", { name: "Search" }));
}

describe("Wa1500Import", () => {
  it("searches only when the button is pressed", async () => {
    const { user, calls } = setup((c) => (c.path === "/api/wa1500/search" ? { body: { results: [hit] } } : undefined));
    await user.type(screen.getByRole("textbox", { name: "Search WA1500" }), "anderson");
    expect(calls).toHaveLength(0);
    await user.click(screen.getByRole("button", { name: "Search" }));
    expect(await screen.findByRole("button", { name: "Import Mike Anderson" })).toBeTruthy();
    expect(calls[0].search).toContain("q=anderson");
  });

  it("focuses the search box and does not search while empty", async () => {
    const { calls } = setup(() => undefined);
    expect(document.activeElement).toBe(screen.getByRole("textbox", { name: "Search WA1500" }));
    expect(calls).toHaveLength(0);
  });

  it("searches on mount when opened with a query", async () => {
    const calls = stubFetch((c) => (c.path === "/api/wa1500/search" ? { body: { results: [hit] } } : undefined));
    render(Wa1500Import, { query: "anderson", onsaved: vi.fn() });
    expect(await screen.findByRole("button", { name: "Import Mike Anderson" })).toBeTruthy();
    expect(calls[0].search).toContain("q=anderson");
  });

  it("searches when Enter is pressed in the box", async () => {
    const { user, calls } = setup((c) => (c.path === "/api/wa1500/search" ? { body: { results: [hit] } } : undefined));
    await user.type(screen.getByRole("textbox", { name: "Search WA1500" }), "anderson{Enter}");
    expect(await screen.findByRole("button", { name: "Import Mike Anderson" })).toBeTruthy();
    expect(calls).toHaveLength(1);
  });

  it("imports and reports the saved shooter", async () => {
    const { user, calls, onsaved } = setup((c) =>
      c.path === "/api/wa1500/search" ? { body: { results: [hit] } } : { status: 201, body: { id: 5, action: "created" } },
    );
    await search(user);
    const importButton = await screen.findByRole("button", { name: "Import Mike Anderson" });
    await user.click(importButton);
    await waitFor(() => expect(onsaved).toHaveBeenCalledWith(expect.objectContaining({ id: 5 })));
    const post = calls.find((c) => c.path === "/api/shooters/import");
    expect(post?.body).toEqual(hit);
  });

  it("lets the admin update a name-matching candidate after a conflict", async () => {
    let imports = 0;
    const { user, calls, onsaved } = setup((c) => {
      if (c.path === "/api/wa1500/search") return { body: { results: [hit] } };
      imports++;
      return imports === 1
        ? { status: 409, body: { error: "name_conflict", candidates: [other] } }
        : { status: 200, body: { id: 9, action: "updated" } };
    });
    await search(user);
    await user.click(screen.getByRole("button", { name: "Import Mike Anderson" }));
    const dialog = await screen.findByRole("dialog", { name: "Name already exists" });
    await user.click(within(dialog).getByRole("button", { name: /Update Mike Anderson · no WAID · Beta/ }));
    await waitFor(() => expect(onsaved).toHaveBeenCalledWith(expect.objectContaining({ id: 9 })));
    const resend = calls.filter((c) => c.path === "/api/shooters/import")[1];
    expect(resend.body).toEqual({ ...hit, update_shooter_id: 9 });
  });

  it("creates a new shooter after a conflict", async () => {
    let imports = 0;
    const { user, calls } = setup((c) => {
      if (c.path === "/api/wa1500/search") return { body: { results: [hit] } };
      imports++;
      return imports === 1
        ? { status: 409, body: { error: "name_conflict", candidates: [other] } }
        : { status: 201, body: { id: 5, action: "created" } };
    });
    await search(user);
    await user.click(screen.getByRole("button", { name: "Import Mike Anderson" }));
    await user.click(await screen.findByRole("button", { name: "Create new shooter" }));
    await waitFor(() => expect(calls.filter((c) => c.path === "/api/shooters/import")).toHaveLength(2));
    expect(calls.filter((c) => c.path === "/api/shooters/import")[1].body).toEqual({ ...hit, create_new: true });
  });

  it("falls back to manual entry when WA1500 is unavailable", async () => {
    const { user, calls, onsaved } = setup((c) =>
      c.path === "/api/wa1500/search" ? { status: 503, body: { error: "wa1500 unavailable" } } : { status: 201, body: { id: 5 } },
    );
    await search(user);
    expect((await screen.findByRole("alert")).textContent).toContain("wa1500 unavailable");
    await user.type(screen.getByRole("textbox", { name: "Last name" }), "anderson");
    await user.type(screen.getByRole("textbox", { name: "First name" }), "mIKE");
    await user.type(screen.getByRole("textbox", { name: "Country" }), "SWE");
    await user.type(screen.getByRole("textbox", { name: "Club" }), "Alpha");
    await user.click(screen.getByRole("button", { name: "Save shooter" }));
    await waitFor(() => expect(onsaved).toHaveBeenCalledWith(expect.objectContaining({ id: 5 })));
    expect(calls.find((c) => c.method === "POST" && c.path === "/api/shooters")?.body).toEqual({
      last_name: "ANDERSON",
      first_name: "Mike",
      country: "SWE",
      club: "Alpha",
      waid: "",
    });
  });

  it("offers manual entry when nothing is found", async () => {
    const { user } = setup(() => ({ body: { results: [] } }));
    await search(user, "nobody");
    expect(await screen.findByText("No WA1500 results")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Enter manually" }));
    expect(screen.getByRole("textbox", { name: "Last name" })).toBeTruthy();
  });
});
