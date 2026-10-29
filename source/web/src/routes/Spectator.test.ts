import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Spectator from "./Spectator.svelte";
import { stubFetch } from "../test/fetch";

const standing = (lane: number, states: string[], withdrawn = false) => ({
  lane,
  category: "Open",
  withdrawn,
  total: 0,
  stages: states.map((state, i) => ({ stage: i + 1, accepted: state === "accepted", state })),
});

let originalFetch: typeof fetch;
let send: ((data: unknown) => void) | null;

function install(status: string, extra?: Parameters<typeof stubFetch>[0]) {
  const calls = stubFetch(
    (call) =>
      extra?.(call) ?? {
        body: {
          matches: [{ id: 7, name: "Club Match", date: "2026-05-01", course: "150", status }],
        },
      },
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
  it("shows the status, stage position and accepted count without progress bars", async () => {
    install("active");
    const { container } = render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({
      current_stage: 2,
      total_stages: 10,
      standings: [
        standing(1, ["accepted", "accepted"]),
        standing(2, ["accepted", "awaiting_acceptance"]),
        standing(3, ["accepted", "in_progress"]),
        standing(4, ["accepted", "in_progress"], true),
      ],
    });
    const row = await screen.findByRole("status");
    await waitFor(() => expect(within(row).getByText("Stage 2 of 10")).toBeTruthy());
    expect(within(row).getByText("1/3 accepted")).toBeTruthy();
    expect(within(row).getByText("active")).toBeTruthy();
    expect(within(row).getByText("Live")).toBeTruthy();
    expect(screen.queryByText("Stage progress")).toBeNull();
    expect(container.querySelector("progress")).toBeNull();
  });

  it("shows Finished instead of the stage position", async () => {
    install("finished");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_stage: 10, total_stages: 10, standings: [standing(1, ["accepted"])] });
    const row = await screen.findByRole("status");
    await waitFor(() => expect(within(row).getByText("Finished")).toBeTruthy());
    expect(within(row).queryByText(/Stage \d+ of/)).toBeNull();
    expect(within(row).queryByText(/accepted/)).toBeNull();
  });

  it("links standings rows to the scorecard only once finished", async () => {
    install("finished");
    const { unmount } = render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_stage: 10, total_stages: 10, standings: [standing(1, ["accepted"])] });
    const link = await screen.findByRole("link", { name: "Scorecard for lane 1" });
    expect(link.getAttribute("href")).toMatch(/^\/scorecard\/\d+\/1$/);
    unmount();
    send = null;
    install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_stage: 2, total_stages: 10, standings: [standing(1, ["accepted"])] });
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
      current_stage: 2,
      total_stages: 10,
      status: "active",
      standings: [standing(1, ["accepted"])],
    });
    await screen.findByRole("table");
    expect(screen.queryByRole("link", { name: /Scorecard for lane/ })).toBeNull();
    send!({
      current_stage: 10,
      total_stages: 10,
      status: "finished",
      standings: [standing(1, ["accepted"])],
    });
    expect(await screen.findByRole("link", { name: "Scorecard for lane 1" })).toBeTruthy();
    expect(loads()).toBe(before);
  });
});

describe("Spectator remote control", () => {
  const key = "matchmark.screen";
  let screenSend: ((data: unknown) => void) | null;
  let screenStatus: number;
  let screenAuth: string | null;

  function installManaged(status = "active") {
    const calls = install(status, (call) =>
      call.method === "POST" && call.path === "/api/spectator/screens"
        ? { status: 201, body: { token: "tok", code: "ABC234" } }
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

  beforeEach(() => {
    localStorage.clear();
    screenSend = null;
    screenStatus = 200;
    screenAuth = null;
  });
  afterEach(() => localStorage.clear());

  it("registers on opt-in, shows the code, stores the token and hides the picker", async () => {
    installManaged();
    render(Spectator);
    await screen.findByRole("combobox");
    await userEvent.click(screen.getByRole("button", { name: "Enable remote control" }));
    expect(await screen.findByText("ABC234")).toBeTruthy();
    expect(JSON.parse(localStorage.getItem(key)!)).toEqual({ token: "tok", code: "ABC234" });
    await waitFor(() => expect(screenAuth).toBe("Bearer tok"));
    expect(screen.queryByRole("combobox")).toBeNull();
    expect(screen.getByText("Waiting for a match")).toBeTruthy();
  });

  it("follows the assigned match and returns to waiting", async () => {
    localStorage.setItem(key, JSON.stringify({ token: "tok", code: "ABC234" }));
    installManaged();
    render(Spectator);
    await waitFor(() => expect(screenSend).not.toBeNull());
    screenSend!({ state: "match", match_id: 7 });
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_stage: 1, total_stages: 4, standings: [standing(1, ["accepted"])] });
    expect(await screen.findByText("Stage 1 of 4")).toBeTruthy();
    expect(screen.queryByRole("combobox")).toBeNull();
    screenSend!({ state: "waiting", match_id: null });
    expect(await screen.findByText("Waiting for a match")).toBeTruthy();
    expect(screen.queryByText("Stage 1 of 4")).toBeNull();
  });

  it("keeps showing the final standings of a finished match", async () => {
    localStorage.setItem(key, JSON.stringify({ token: "tok", code: "ABC234" }));
    installManaged("finished");
    render(Spectator);
    await waitFor(() => expect(screenSend).not.toBeNull());
    screenSend!({ state: "match", match_id: 7 });
    await waitFor(() => expect(send).not.toBeNull());
    send!({
      status: "finished",
      current_stage: 4,
      total_stages: 4,
      standings: [standing(1, ["accepted"])],
    });
    expect(await screen.findByText("Finished")).toBeTruthy();
  });

  it("leaves remote control and reverts to the match picker", async () => {
    localStorage.setItem(key, JSON.stringify({ token: "tok", code: "ABC234" }));
    const calls = installManaged();
    render(Spectator);
    await userEvent.click(await screen.findByRole("button", { name: "Leave remote control" }));
    expect(await screen.findByRole("combobox")).toBeTruthy();
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
    render(Spectator);
    expect(await screen.findByRole("alert")).toBeTruthy();
    expect(await screen.findByRole("combobox")).toBeTruthy();
    expect(localStorage.getItem(key)).toBeNull();
  });
});
