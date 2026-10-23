import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import Scorer from "./Scorer.svelte";
import { stubFetch } from "../test/fetch";

type SessionOverrides = {
  score_lane?: number;
  shooter_name?: string;
  phase?: "scoring" | "waiting" | "needs_acceptance" | "next_stage" | "finished";
  current_stage?: number;
  acceptance?: object;
  disputed?: number[];
  counts?: Record<string, number>;
};

const zeroCounts = { x: 0, 10: 0, 9: 0, 8: 0, 7: 0, 0: 0, miss: 0 };

const session = (locked: boolean[], { disputed = [], counts = {}, ...overrides }: SessionOverrides = {}) => ({
  lane: 1,
  score_lane: 2,
  current_stage: locked.findIndex((l) => !l) + 1 || locked.length,
  entry_id: 4,
  shooter_name: "Rick Roe",
  phase: "scoring",
  stages: locked.map((l, i) => ({
    ...zeroCounts,
    ...(disputed.includes(i + 1) ? counts : {}),
    stage_number: i + 1,
    expected_shots: 30,
    status: l ? "awaiting_acceptance" : "open",
    disputed: disputed.includes(i + 1),
  })),
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
    expect(screen.getByText(/Stage 2\./)).toBeTruthy();
    expect(screen.queryByRole("tab")).toBeNull();
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
    await open(server, session([true, false], { current_stage: 1 }));
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
    server.streams[0].send("snapshot", session([true, true], { phase: "waiting", current_stage: 2 }));
    await screen.findByText(/Stage 2 is locked/);
    expect(screen.queryByRole("button", { name: "Save" })).toBeNull();
    server.streams[0].send("snapshot", session([true, false]));
    await screen.findByRole("button", { name: "Save" });
    expect(screen.getByLabelText("X count")).toHaveProperty("value", "1");
  });

  it("shows the waiting screen with a banner and a button when the own stage awaits acceptance", async () => {
    const server = fakeStream();
    const user = userEvent.setup();
    const acceptance = { ...zeroCounts, x: 10, 10: 2, stage_number: 1, total: 12, scorer_name: "Ann Lee" };
    await open(server, session([true, false], { phase: "needs_acceptance", acceptance, current_stage: 1 }));
    expect(screen.getByText(/Your result for stage 1 awaits your acceptance/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save" })).toBeNull();

    await user.click(screen.getByRole("button", { name: "Review result" }));
    expect(screen.getByText("Scored by Ann Lee")).toBeTruthy();
    expect(screen.getByText(/12 hits, 10 X/)).toBeTruthy();
    expect(screen.queryByLabelText("X count")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Accept" }));
    const accept = calls.find((c) => c.method === "POST");
    expect(accept?.path).toBe("/api/scorer/stages/1/accept");
    expect(accept?.authorization).toBe("Bearer device-token");
  });

  it("disputes from the result view", async () => {
    const server = fakeStream();
    const user = userEvent.setup();
    const acceptance = { ...zeroCounts, x: 12, stage_number: 2, total: 12, scorer_name: "Ann Lee" };
    await open(server, session([false, true], { phase: "needs_acceptance", acceptance }));
    await user.click(screen.getByRole("button", { name: "Review result" }));
    await user.click(screen.getByRole("button", { name: "Dispute" }));
    expect(calls.find((c) => c.method === "POST")?.path).toBe("/api/scorer/stages/2/dispute");
  });

  it("shows a persistent disputed notice with the previous values prefilled", async () => {
    const server = fakeStream();
    await open(server, session([false, false], { disputed: [1], counts: { x: 7, 10: 5 } }));
    expect(screen.getByText(/Disputed: the shooter disputed stage 1/)).toBeTruthy();
    expect(screen.getByLabelText("X count")).toHaveProperty("value", "7");
    expect(screen.getByLabelText("10 count")).toHaveProperty("value", "5");
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

  it("announces a started stage and shows the scoring form after Continue", async () => {
    const server = fakeStream();
    const user = userEvent.setup();
    await open(server, session([true, false], { phase: "next_stage" }));
    expect(screen.getByText(/The next stage has started/)).toBeTruthy();
    expect(screen.queryByLabelText("X count")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByLabelText("X count")).toBeTruthy();
  });

  it("shows the finished screen and keeps the token", async () => {
    const server = fakeStream();
    render(Scorer);
    await waitFor(() => expect(server.streams.length).toBe(1));
    server.streams[0].send("snapshot", { lane: 1, score_lane: 0, current_stage: 0, entry_id: 0, shooter_name: "", stages: [], phase: "finished" });
    await screen.findByText(/The competition is finished/);
    expect(localStorage.getItem("mm_device_token")).toBe("device-token");
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
