import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createSseParser, openStream, type StreamState } from "./sse";

describe("createSseParser", () => {
  it("emits events split across chunks and ignores comments", () => {
    const seen: [string, string][] = [];
    const parser = createSseParser((event, data) => seen.push([event, data]));
    parser.push(": heartbeat\n\nevent: snap");
    parser.push("shot\ndata: {\"a\":1}\n\nevent: other\r\ndata: x\r\ndata: y\r\n\r\n");
    expect(seen).toEqual([
      ["snapshot", '{"a":1}'],
      ["other", "x\ny"],
    ]);
  });

  it("defaults the event name to message", () => {
    const seen: string[] = [];
    createSseParser((event) => seen.push(event)).push("data: 1\n\n");
    expect(seen).toEqual(["message"]);
  });
});

type Conn = { url: string; headers: Headers; push(text: string): void; end(): void };

function fakeServer() {
  const conns: Conn[] = [];
  const statuses: number[] = [];
  const retryAfter: string[] = [];
  const fetchFn = ((input: RequestInfo | URL, init?: RequestInit) => {
    const status = statuses.shift() ?? 200;
    if (status !== 200) {
      const headers: Record<string, string> = {};
      const after = retryAfter.shift();
      if (after) headers["Retry-After"] = after;
      return Promise.resolve(new Response(null, { status, headers }));
    }
    let controller!: ReadableStreamDefaultController<Uint8Array>;
    const body = new ReadableStream<Uint8Array>({ start: (c) => (controller = c) });
    const enc = new TextEncoder();
    init?.signal?.addEventListener("abort", () => {
      try {
        controller.error(new DOMException("aborted", "AbortError"));
      } catch {
        return;
      }
    });
    conns.push({
      url: String(input),
      headers: new Headers(init?.headers),
      push: (text) => controller.enqueue(enc.encode(text)),
      end: () => controller.close(),
    });
    return Promise.resolve(new Response(body, { status: 200, headers: { "Content-Type": "text/event-stream" } }));
  }) as typeof fetch;
  return { conns, statuses, retryAfter, fetchFn };
}

const tick = (ms = 0) => vi.advanceTimersByTimeAsync(ms);

function setHidden(hidden: boolean) {
  Object.defineProperty(document, "hidden", { configurable: true, get: () => hidden });
  document.dispatchEvent(new Event("visibilitychange"));
}

describe("openStream", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    setHidden(false);
  });

  function open(server: ReturnType<typeof fakeServer>, extra: Partial<Parameters<typeof openStream>[0]> = {}) {
    const events: [string, string][] = [];
    const states: StreamState[] = [];
    const handle = openStream({
      url: "/api/stream",
      headers: { Authorization: "Bearer t" },
      fetch: server.fetchFn,
      random: () => 1,
      onEvent: (event, data) => events.push([event, data]),
      onState: (s) => states.push(s),
      ...extra,
    });
    return { events, states, handle };
  }

  it("delivers events with the supplied headers and goes live", async () => {
    const server = fakeServer();
    const { events, states } = open(server);
    await tick();
    server.conns[0].push('event: snapshot\ndata: {"n":1}\n\n');
    await tick();
    expect(server.conns[0].headers.get("Authorization")).toBe("Bearer t");
    expect(server.conns[0].headers.get("Accept")).toBe("text/event-stream");
    expect(events).toEqual([["snapshot", '{"n":1}']]);
    expect(states).toEqual(["connecting", "live"]);
  });

  it("goes stale and reconnects with growing backoff, resetting after success", async () => {
    const server = fakeServer();
    server.statuses.push(200, 502, 502);
    const { states } = open(server, { backoff: { initialMs: 1000, maxMs: 30000 } });
    await tick();
    server.conns[0].end();
    await tick();
    expect(states.at(-1)).toBe("stale");
    await tick(1000);
    expect(server.statuses).toHaveLength(1);
    await tick(1999);
    expect(server.statuses).toHaveLength(1);
    await tick(1);
    expect(server.statuses).toHaveLength(0);
    await tick(3999);
    expect(server.conns).toHaveLength(1);
    await tick(1);
    expect(server.conns).toHaveLength(2);
    expect(states.at(-1)).toBe("live");
  });

  it("waits at least the Retry-After of a 503 before reconnecting", async () => {
    const server = fakeServer();
    server.statuses.push(503);
    server.retryAfter.push("5");
    open(server, { backoff: { initialMs: 100, maxMs: 1000 } });
    await tick();
    await tick(4999);
    expect(server.conns).toHaveLength(0);
    await tick(1);
    expect(server.conns).toHaveLength(1);
  });

  it("stops without retrying on a terminal revoked event", async () => {
    const server = fakeServer();
    const { states } = open(server);
    await tick();
    server.conns[0].push("event: revoked\ndata: {}\n\n");
    await tick(60000);
    expect(states.at(-1)).toBe("ended");
    expect(server.conns).toHaveLength(1);
  });

  it("keeps backing off when connections close immediately", async () => {
    const server = fakeServer();
    open(server, { backoff: { initialMs: 1000, maxMs: 30000 } });
    await tick();
    server.conns[0].end();
    await tick(1000);
    expect(server.conns).toHaveLength(2);
    server.conns[1].end();
    await tick(1999);
    expect(server.conns).toHaveLength(2);
    await tick(1);
    expect(server.conns).toHaveLength(3);
  });

  it("does not retry on 401, 403, 404, 409 or 410", async () => {
    const server = fakeServer();
    server.statuses.push(410);
    const { states } = open(server);
    await tick(60000);
    expect(states.at(-1)).toBe("ended");
    expect(server.conns).toHaveLength(0);
  });

  it("closes while the tab is hidden and reconnects when visible again", async () => {
    const server = fakeServer();
    const { states } = open(server);
    await tick();
    setHidden(true);
    await tick(60000);
    expect(states.at(-1)).toBe("paused");
    expect(server.conns).toHaveLength(1);
    setHidden(false);
    await tick();
    expect(server.conns).toHaveLength(2);
    expect(states.at(-1)).toBe("live");
  });

  it("does not connect at all while the tab starts hidden", async () => {
    setHidden(true);
    const server = fakeServer();
    open(server);
    await tick(5000);
    expect(server.conns).toHaveLength(0);
  });

  it("reconnects when the server goes silent past the stale timeout", async () => {
    const server = fakeServer();
    const { states } = open(server, { staleAfterMs: 5000, backoff: { initialMs: 100, maxMs: 1000 } });
    await tick();
    await tick(5000);
    expect(states).toContain("stale");
    await tick(100);
    expect(server.conns).toHaveLength(2);
  });

  it("stops for good when closed and never polls", async () => {
    const server = fakeServer();
    const { handle, states } = open(server);
    await tick();
    handle.close();
    await tick(120000);
    expect(server.conns).toHaveLength(1);
    expect(states.at(-1)).toBe("ended");
  });
});
