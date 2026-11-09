import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Spectator from "./Spectator.svelte";
import SiteHeader from "../lib/SiteHeader.svelte";
import { screenControl } from "../lib/screenControl.svelte";
import { router } from "../lib/router.svelte";
import { stubFetch } from "../test/fetch";

const standing = (lane: number, states: string[], withdrawn = false) => ({
  lane,
  category: "Open",
  withdrawn,
  total: 0,
  segments: states.map((state, i) => ({ segment: i + 1, accepted: state === "accepted", state })),
});

let originalFetch: typeof fetch;
let send: ((data: unknown) => void) | null;

function install(status: string, extra?: Parameters<typeof stubFetch>[0]) {
  const calls = stubFetch(
    (call) =>
      extra?.(call) ??
      (call.path === "/api/rulebook"
        ? {
            body: {
              courses: [
                {
                  key: "150",
                  label: "150",
                  discipline: "1500",
                  total_shots: 150,
                  categories: ["Revolver 1500", "Pistol 1500"],
                  stages: [],
                  segments: [
                    { number: 1, name: "Match 1", stages: [1, 2], shots: 24 },
                    { number: 2, name: "Match 5 · Stage 1+2", stages: [7, 8], shots: 30 },
                  ],
                },
              ],
              rings: [],
            },
          }
        : {
            body: {
              matches: [{ id: 7, name: "Club Match", date: "2026-05-01", course: "150", status }],
            },
          }),
  );
  const previous = globalThis.fetch;
  globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(input, init);
    if (new URL(request.url).pathname !== "/api/spectator/matches/7/stream")
      return previous(input, init);
    const enc = new TextEncoder();
    let controller!: ReadableStreamDefaultController<Uint8Array>;
    const body = new ReadableStream<Uint8Array>({ start: (c) => (controller = c) });
    send = (data) =>
      controller.enqueue(enc.encode(`event: snapshot\ndata: ${JSON.stringify(data)}\n\n`));
    return Promise.resolve(
      new Response(body, { headers: { "Content-Type": "text/event-stream" } }),
    );
  }) as typeof fetch;
  return calls;
}

beforeEach(() => {
  originalFetch = globalThis.fetch;
  send = null;
});
afterEach(() => {
  globalThis.fetch = originalFetch;
});

describe("Spectator status row", () => {
  it("shows the status, segment position and accepted count without progress bars", async () => {
    install("active");
    const { container } = render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({
      current_segment: 2,
      total_segments: 10,
      standings: [
        standing(1, ["accepted", "accepted"]),
        standing(2, ["accepted", "awaiting_acceptance"]),
        standing(3, ["accepted", "in_progress"]),
        standing(4, ["accepted", "in_progress"], true),
      ],
    });
    const row = await screen.findByRole("status");
    await waitFor(() => expect(within(row).getByText("Segment 2 of 10")).toBeTruthy());
    expect(within(row).getByText("1/3 accepted")).toBeTruthy();
    expect(within(row).getByText("active")).toBeTruthy();
    expect(within(row).getByText("Live")).toBeTruthy();
    expect(screen.queryByText("Segment progress")).toBeNull();
    expect(container.querySelector("progress")).toBeNull();
  });

  it("heads each segment column with its short score-card name", async () => {
    install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({
      current_segment: 1,
      total_segments: 2,
      standings: [standing(1, ["in_progress", "not_started"])],
    });
    expect(await screen.findByRole("columnheader", { name: "M1" })).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "Stage 1+2" })).toBeTruthy();
  });

  it("shows a Classification column with the Classification name, hiding Unclassified", async () => {
    install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    const named = (lane: number, last: string, classification?: object) => ({
      ...standing(lane, ["in_progress"]),
      first_name: "A",
      last_name: last,
      classification,
    });
    send!({
      current_segment: 1,
      total_segments: 2,
      standings: [
        named(1, "Berg", { value: 4, name: "Master" }),
        named(2, "Dahl", { value: 0, name: "Unclassified" }),
      ],
    });
    expect(await screen.findByRole("columnheader", { name: "Classification" })).toBeTruthy();
    const cells = (name: RegExp) =>
      within(screen.getByRole("row", { name }))
        .getAllByRole("cell")
        .map((c) => c.textContent);
    expect(cells(/Berg/)).toContain("Master");
    expect(cells(/Dahl/)).not.toContain("Unclassified");
  });

  it("omits the Classification column when no standing has a Classification", async () => {
    install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_segment: 1, total_segments: 2, standings: [standing(1, ["in_progress"])] });
    await screen.findByRole("columnheader", { name: "Total" });
    expect(screen.queryByRole("columnheader", { name: "Classification" })).toBeNull();
  });

  it("shows Finished instead of the segment position", async () => {
    install("finished");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_segment: 10, total_segments: 10, standings: [standing(1, ["accepted"])] });
    const row = await screen.findByRole("status");
    await waitFor(() => expect(within(row).getByText("Finished")).toBeTruthy());
    expect(within(row).queryByText(/Segment \d+ of/)).toBeNull();
    expect(within(row).queryByText(/accepted/)).toBeNull();
  });

  it("links standings rows to the scorecard only once finished", async () => {
    install("finished");
    const { unmount } = render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_segment: 10, total_segments: 10, standings: [standing(1, ["accepted"])] });
    const link = await screen.findByRole("link", { name: "Scorecard for lane 1" });
    expect(link.getAttribute("href")).toMatch(/^\/scorecard\/\d+\/1$/);
    unmount();
    send = null;
    install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_segment: 2, total_segments: 10, standings: [standing(1, ["accepted"])] });
    await screen.findByRole("table");
    expect(screen.queryByRole("link", { name: /Scorecard for lane/ })).toBeNull();
  });

  it("notices a match finishing from the snapshot without reloading the list", async () => {
    const calls = install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    const loads = () => calls.filter((c) => c.path === "/api/spectator/matches").length;
    const before = loads();
    send!({
      current_segment: 2,
      total_segments: 10,
      status: "active",
      standings: [standing(1, ["accepted"])],
    });
    await screen.findByRole("table");
    expect(screen.queryByRole("link", { name: /Scorecard for lane/ })).toBeNull();
    send!({
      current_segment: 10,
      total_segments: 10,
      status: "finished",
      standings: [standing(1, ["accepted"])],
    });
    expect(await screen.findByRole("link", { name: "Scorecard for lane 1" })).toBeTruthy();
    expect(loads()).toBe(before);
  });
});

describe("Spectator standings grouping", () => {
  const entry = (lane: number, category: string, total: number, x: number, withdrawn = false) => ({
    lane,
    category,
    withdrawn,
    total,
    x_count: x,
    segments: [],
  });
  const rows = [
    entry(1, "Pistol 1500", 100, 2),
    entry(2, "Revolver 1500", 90, 1),
    entry(3, "Revolver 1500", 90, 5),
    entry(4, "Revolver 1500", 10, 0, true),
    entry(5, "Revolver 1500", 70, 0),
    entry(6, "Pistol 1500", 100, 2),
  ];

  async function open(group: boolean, standings: unknown[]) {
    install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_segment: 1, total_segments: 2, group_by_category: group, standings });
  }

  const cells = (table: HTMLElement) =>
    within(table)
      .getAllByRole("row")
      .slice(1)
      .map((r) => [...r.querySelectorAll("td")].slice(0, 2).map((c) => c.textContent));

  it("splits a category into ranked Class groups when standings carry a class group", async () => {
    const classed = (lane: number, total: number, value: number, name: string) => ({
      ...entry(lane, "Revolver 1500", total, 0),
      class_group: { value, name },
    });
    await open(true, [
      classed(1, 90, 4, "Master"),
      classed(2, 80, 4, "Master"),
      classed(3, 70, 1, "Marksman"),
      classed(4, 99, 5, "High Master"),
    ]);
    await screen.findByRole("heading", { name: "Revolver 1500 - High Master" });
    const headings = screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent?.trim());
    expect(headings).toEqual([
      "Revolver 1500 - High Master",
      "Revolver 1500 - Master",
      "Revolver 1500 - Marksman",
    ]);
    const tables = screen.getAllByRole("table");
    expect(cells(tables[1])).toEqual([
      ["1", "1"],
      ["2", "2"],
    ]);
    expect(cells(tables[2])).toEqual([["1", "3"]]);
  });

  it("renders one ranked table per category in rulebook order with withdrawn last", async () => {
    await open(true, rows);
    await screen.findByRole("heading", { name: "Revolver 1500" });
    const headings = screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent);
    expect(headings).toEqual(["Revolver 1500", "Pistol 1500"]);
    expect(screen.getByRole("heading", { name: "Standings" }).classList).not.toContain("sr-only");
    const [revolver, pistol] = screen.getAllByRole("table");
    expect(cells(revolver)).toEqual([
      ["1", "3"],
      ["2", "2"],
      ["3", "5"],
      ["", "4"],
    ]);
    expect(cells(pistol)).toEqual([
      ["1", "1"],
      ["1", "6"],
    ]);
  });

  it("keeps one combined table with overall ranks when not grouped", async () => {
    await open(false, [entry(1, "Pistol 1500", 100, 2), entry(2, "Revolver 1500", 90, 1)]);
    await screen.findByRole("heading", { name: "Standings" });
    expect(screen.getAllByRole("table")).toHaveLength(1);
    expect(screen.queryAllByRole("heading", { level: 3 })).toHaveLength(0);
    expect(cells(screen.getByRole("table"))).toEqual([
      ["1", "1"],
      ["2", "2"],
    ]);
  });

  it("puts withdrawn entries last and unranked when not grouped", async () => {
    await open(false, [
      entry(3, "Pistol 1500", 100, 2),
      entry(1, "Pistol 1500", 95, 0, true),
      entry(2, "Revolver 1500", 90, 1),
    ]);
    await screen.findByRole("table");
    expect(cells(screen.getByRole("table"))).toEqual([
      ["1", "3"],
      ["2", "2"],
      ["", "1"],
    ]);
  });

  it("shows a single category without a heading in either mode", async () => {
    await open(true, [entry(1, "Revolver 1500", 100, 2), entry(2, "Revolver 1500", 90, 1)]);
    await screen.findByRole("table");
    expect(screen.queryAllByRole("heading", { level: 3 })).toHaveLength(0);
    expect(screen.getByRole("heading", { name: "Standings" }).classList).toContain("sr-only");
    expect(screen.getAllByRole("table")).toHaveLength(1);
  });
});

describe("Spectator remote control", () => {
  const key = "matchmark.screen";
  let screenSend: ((data: unknown) => void) | null;
  let screenStatus: number;
  let screenAuth: string | null;

  function installManaged(status = "active", opts = { rejectEnable: false }) {
    const calls = install(status, (call) =>
      call.method === "POST" && call.path === "/api/spectator/screens"
        ? opts.rejectEnable
          ? { status: 500, body: { error: "Nope" } }
          : { status: 201, body: { token: "tok", code: "ABC234" } }
        : call.method === "DELETE"
          ? { status: 204 }
          : undefined,
    );
    const previous = globalThis.fetch;
    globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init);
      if (new URL(request.url).pathname !== "/api/spectator/screens/stream")
        return previous(input, init);
      screenAuth = request.headers.get("Authorization");
      if (screenStatus !== 200)
        return Promise.resolve(new Response(null, { status: screenStatus }));
      const enc = new TextEncoder();
      let controller!: ReadableStreamDefaultController<Uint8Array>;
      const body = new ReadableStream<Uint8Array>({ start: (c) => (controller = c) });
      screenSend = (data) =>
        controller.enqueue(enc.encode(`event: snapshot\ndata: ${JSON.stringify(data)}\n\n`));
      return Promise.resolve(
        new Response(body, { headers: { "Content-Type": "text/event-stream" } }),
      );
    }) as typeof fetch;
    return calls;
  }

  function renderPage() {
    screenControl.stop();
    const stored = localStorage.getItem(key);
    screenControl.registration = stored ? JSON.parse(stored) : null;
    screenControl.assignment = null;
    screenControl.connection = "connecting";
    screenControl.notice = "";
    screenControl.error = "";
    screenControl.busy = false;
    render(SiteHeader);
    render(Spectator);
  }

  async function menuItem(name: string) {
    const trigger = screen.getByRole("button", { name: "Menu" });
    if (trigger.getAttribute("aria-expanded") !== "true") await userEvent.click(trigger);
    return screen.findByRole("button", { name });
  }

  beforeEach(() => {
    localStorage.clear();
    router.path = "/";
    screenSend = null;
    screenStatus = 200;
    screenAuth = null;
  });
  afterEach(() => localStorage.clear());

  it("registers on opt-in, shows the code, stores the token and hides the picker", async () => {
    installManaged();
    renderPage();
    await screen.findByRole("combobox");
    await userEvent.click(await menuItem("Enable remote control"));
    expect(await screen.findByText("ABC234")).toBeTruthy();
    expect(JSON.parse(localStorage.getItem(key)!)).toEqual({ token: "tok", code: "ABC234" });
    await waitFor(() => expect(screenAuth).toBe("Bearer tok"));
    expect(screen.queryByRole("combobox")).toBeNull();
    expect(screen.getByText("Waiting for a match")).toBeTruthy();
  });

  it("follows the assigned match and returns to waiting", async () => {
    localStorage.setItem(key, JSON.stringify({ token: "tok", code: "ABC234" }));
    installManaged();
    renderPage();
    await waitFor(() => expect(screenSend).not.toBeNull());
    screenSend!({ state: "match", match_id: 7, approved: true });
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_segment: 1, total_segments: 4, standings: [standing(1, ["accepted"])] });
    expect(await screen.findByText("Segment 1 of 4")).toBeTruthy();
    expect(screen.queryByRole("combobox")).toBeNull();
    screenSend!({ state: "waiting", match_id: null, approved: true });
    expect(await screen.findByText("Waiting for a match")).toBeTruthy();
    expect(screen.queryByText("Segment 1 of 4")).toBeNull();
  });

  it("keeps showing the final standings of a finished match", async () => {
    localStorage.setItem(key, JSON.stringify({ token: "tok", code: "ABC234" }));
    installManaged("finished");
    renderPage();
    await waitFor(() => expect(screenSend).not.toBeNull());
    screenSend!({ state: "match", match_id: 7, approved: true });
    await waitFor(() => expect(send).not.toBeNull());
    send!({
      status: "finished",
      current_segment: 4,
      total_segments: 4,
      standings: [standing(1, ["accepted"])],
    });
    expect(await screen.findByText("Finished")).toBeTruthy();
  });

  it("shows the code until approved, then offers leave below the standings", async () => {
    localStorage.setItem(key, JSON.stringify({ token: "tok", code: "ABC234" }));
    installManaged();
    renderPage();
    await waitFor(() => expect(screenSend).not.toBeNull());
    screenSend!({ state: "waiting", match_id: null, approved: false });
    expect(await screen.findByRole("region", { name: "Remote control" })).toBeTruthy();
    screenSend!({ state: "waiting", match_id: null, approved: true });
    await waitFor(() =>
      expect(screen.queryByRole("region", { name: "Remote control" })).toBeNull(),
    );
    expect(await menuItem("Leave remote control")).toBeTruthy();
  });

  it("closes the menu when the item is clicked", async () => {
    installManaged();
    renderPage();
    await screen.findByRole("combobox");
    await userEvent.click(await menuItem("Enable remote control"));
    expect(screen.queryByRole("button", { name: "Enable remote control" })).toBeNull();
  });

  it("shows enable errors in the notice card", async () => {
    installManaged("active", { rejectEnable: true });
    renderPage();
    await screen.findByRole("combobox");
    await userEvent.click(await menuItem("Enable remote control"));
    expect((await screen.findByRole("alert")).textContent).toContain("Nope");
  });

  it("offers no remote control on other routes", async () => {
    router.path = "/results";
    installManaged();
    renderPage();
    await userEvent.click(screen.getByRole("button", { name: "Menu" }));
    expect(screen.queryByRole("button", { name: "Enable remote control" })).toBeNull();
  });

  it("leaves remote control and reverts to the match picker", async () => {
    localStorage.setItem(key, JSON.stringify({ token: "tok", code: "ABC234" }));
    const calls = installManaged();
    renderPage();
    await userEvent.click(await menuItem("Leave remote control"));
    expect(await screen.findByRole("combobox")).toBeTruthy();
    expect((await menuItem("Enable remote control")).textContent).toBeTruthy();
    expect(localStorage.getItem(key)).toBeNull();
    const leave = calls.find((c) => c.method === "DELETE");
    expect(leave?.path).toBe("/api/spectator/screens/self");
    expect(leave?.authorization).toBe("Bearer tok");
  });

  it.each([401, 410])("shows a notice and reverts when the stream answers %i", async (status) => {
    localStorage.setItem(key, JSON.stringify({ token: "tok", code: "ABC234" }));
    screenStatus = status;
    installManaged();
    screenStatus = status;
    renderPage();
    expect(await screen.findByRole("alert")).toBeTruthy();
    expect(await screen.findByRole("combobox")).toBeTruthy();
    expect(localStorage.getItem(key)).toBeNull();
  });
});
