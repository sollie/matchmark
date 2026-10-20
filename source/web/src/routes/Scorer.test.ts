import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Scorer from "./Scorer.svelte";
import { stubFetch } from "../test/fetch";

type SessionOverrides = { score_lane?: number; shooter_name?: string };

const session = (locked: boolean[], overrides: SessionOverrides = {}) => ({
  lane: 1,
  score_lane: 2,
  entry_id: 4,
  shooter_name: "Rick Roe",
  stages: locked.map((l, i) => ({ stage_number: i + 1, expected_shots: 30, locked: l })),
  ...overrides,
});

type Server = {
  streams: { headers: Headers; send(event: string, data: unknown): void; end(): void }[];
  respondStatus?: number;
};

function fakeStream(): Server {
  const server: Server = { streams: [] };
  const previous = globalThis.fetch;
  globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(input, init);
    if (new URL(request.url).pathname !== "/api/scorer/stream") return previous(input, init);
    if (server.respondStatus) return Promise.resolve(new Response(null, { status: server.respondStatus }));
    let controller!: ReadableStreamDefaultController<Uint8Array>;
    const body = new ReadableStream<Uint8Array>({ start: (c) => (controller = c) });
    const enc = new TextEncoder();
    server.streams.push({
      headers: request.headers,
      send: (event, data) => controller.enqueue(enc.encode(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`)),
      end: () => controller.close(),
    });
    return Promise.resolve(new Response(body, { status: 200, headers: { "Content-Type": "text/event-stream" } }));
  }) as typeof fetch;
  return server;
}

async function open(server: Server, snapshot = session([true, false])) {
  render(Scorer);
  await waitFor(() => expect(server.streams.length).toBe(1));
  server.streams[0].send("snapshot", snapshot);
  await screen.findByText(`Scoring lane ${snapshot.score_lane}: ${snapshot.shooter_name}`);
}

let calls: ReturnType<typeof stubFetch>;
let originalFetch: typeof fetch;

beforeEach(() => {
  localStorage.setItem("mm_device_token", "device-token");
  originalFetch = globalThis.fetch;
  calls = stubFetch();
});
afterEach(() => {
  globalThis.fetch = originalFetch;
  vi.restoreAllMocks();
  localStorage.clear();
});

describe("Scorer", () => {
  it("opens the stream with the device token and saves counts for the first open stage", async () => {
    const server = fakeStream();
    const user = userEvent.setup();
    await open(server);
    expect(server.streams[0].headers.get("Authorization")).toBe("Bearer device-token");
    expect(screen.getByRole("tab", { name: /Stage 2/ }).getAttribute("aria-selected")).toBe("true");
    await user.click(screen.getByRole("button", { name: "Increase X" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    const save = calls.find((c) => c.method === "PUT");
    expect(save?.path).toBe("/api/scorer/stages/2");
    expect(save?.authorization).toBe("Bearer device-token");
    expect(save?.body).toMatchObject({ x: 1 });
  });

  it("shows a locked banner for a locked stage and saves nothing", async () => {
    const server = fakeStream();
    const user = userEvent.setup();
    await open(server);
    await user.click(screen.getByRole("tab", { name: /Stage 1/ }));
    expect(screen.getByText(/Stage 1 is locked/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save" })).toBeNull();
    expect(calls.some((c) => c.method === "PUT")).toBe(false);
  });

  it("keeps unsaved ring counts when the session updates", async () => {
    const server = fakeStream();
    const user = userEvent.setup();
    await open(server);
    await user.click(screen.getByRole("button", { name: "Increase X" }));
    server.streams[0].send("snapshot", session([true, false]));
    server.streams[0].send("snapshot", session([true, false]));
    await waitFor(() => expect(screen.getByLabelText("X count")).toHaveProperty("value", "1"));
    expect(screen.queryByText(/ring counts were reset/i)).toBeNull();
  });

  it("disables saving when the current stage becomes locked and restores it when unlocked", async () => {
    const server = fakeStream();
    const user = userEvent.setup();
    await open(server);
    await user.click(screen.getByRole("button", { name: "Increase X" }));
    server.streams[0].send("snapshot", session([true, true]));
    await screen.findByText(/Stage 2 is locked/);
    expect(screen.queryByRole("button", { name: "Save" })).toBeNull();
    server.streams[0].send("snapshot", session([true, false]));
    await screen.findByRole("button", { name: "Save" });
    expect(screen.getByLabelText("X count")).toHaveProperty("value", "1");
  });

  it("shows a notice and resets the counts when the score lane changes", async () => {
    const server = fakeStream();
    const user = userEvent.setup();
    await open(server);
    await user.click(screen.getByRole("button", { name: "Increase X" }));
    server.streams[0].send("snapshot", session([true, false], { score_lane: 3, shooter_name: "Ann Lee" }));
    await screen.findByText("Scoring lane 3: Ann Lee");
    expect(screen.getByRole("status").textContent).toContain("Score lane changed to lane 3");
    expect(screen.getByLabelText("X count")).toHaveProperty("value", "0");
  });

  it("forgets the token and does not reconnect after a revoked event", async () => {
    const server = fakeStream();
    await open(server);
    server.streams[0].send("revoked", {});
    await screen.findByText("Open a pairing link from the admin to score a lane.");
    expect(localStorage.getItem("mm_device_token")).toBeNull();
    await new Promise((r) => setTimeout(r, 50));
    expect(server.streams.length).toBe(1);
  });

  it("forgets the token when the stream is rejected as unauthorized", async () => {
    const server = fakeStream();
    server.respondStatus = 401;
    calls = stubFetch((c) => (c.path === "/api/scorer/session" ? { status: 401 } : undefined));
    render(Scorer);
    await screen.findByText("Open a pairing link from the admin to score a lane.");
    expect(localStorage.getItem("mm_device_token")).toBeNull();
  });
});
