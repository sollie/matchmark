import { describe, expect, it, vi } from "vitest";
import { MatchPage, type MatchPageDeps } from "./matchPage.svelte";
import { stubFetch, type Call } from "../test/fetch";
import type { StreamOptions } from "./sse";

const match = (status = "draft") => ({ id: 5, name: "Spring Open", date: "2026-05-01", course: "150", status });
const entry = { id: 1, shooter_id: 7, lane: 1, stages: [] };
const shooter = { id: 7, first_name: "Jane", last_name: "Doe" };

function setup(respond: (call: Call) => { status?: number; body?: unknown } | undefined = () => undefined) {
  const calls = stubFetch((call) => {
    const custom = respond(call);
    if (custom) return custom;
    if (call.path === "/api/matches/5" && call.method === "GET") return { body: match() };
    if (call.path === "/api/matches/5/entries") return { body: { entries: [entry] } };
    if (call.path === "/api/shooters") return { body: { shooters: [shooter] } };
    return undefined;
  });
  const streams: { options: StreamOptions; close: ReturnType<typeof vi.fn> }[] = [];
  const deps: MatchPageDeps = {
    openStream: (options) => {
      const close = vi.fn();
      streams.push({ options, close });
      return { close };
    },
    navigate: vi.fn(),
  };
  return { page: new MatchPage(5, deps), calls, streams, deps };
}

const paths = (calls: Call[]) => calls.map((c) => `${c.method} ${c.path}`);

describe("MatchPage", () => {
  it("loads the match and shooters and subscribes to the stream", async () => {
    const { page, calls, streams } = setup();
    const stop = page.start();
    await vi.waitFor(() => expect(page.match?.name).toBe("Spring Open"));
    await vi.waitFor(() => expect(page.shooters).toHaveLength(1));
    expect(paths(calls)).toEqual(expect.arrayContaining(["GET /api/matches/5", "GET /api/shooters"]));
    expect(streams[0].options.url).toBe("/api/matches/5/stream");
    stop();
    expect(streams[0].close).toHaveBeenCalled();
  });

  it("shows the server error when the match cannot be loaded", async () => {
    const { page } = setup((call) =>
      call.path === "/api/matches/5" ? { status: 404, body: { error: "match not found" } } : undefined,
    );
    page.start();
    await vi.waitFor(() => expect(page.error).toBe("match not found"));
    expect(page.match).toBeNull();
  });

  it("refetches entries and the match when a changed event arrives", async () => {
    const { page, calls, streams } = setup();
    page.start();
    await vi.waitFor(() => expect(page.match).not.toBeNull());
    calls.length = 0;
    streams[0].options.onEvent("snapshot", "");
    expect(calls).toHaveLength(0);
    streams[0].options.onEvent("changed", "");
    await vi.waitFor(() => expect(page.entries).toHaveLength(1));
    expect(paths(calls)).toEqual(expect.arrayContaining(["GET /api/matches/5", "GET /api/matches/5/entries"]));
    expect(page.version).toBe(1);
  });

  it("moves to the next status and keeps the other fields", async () => {
    const { page, calls } = setup((call) =>
      call.path === "/api/matches/5/status" ? { body: { status: "active" } } : undefined,
    );
    page.start();
    await vi.waitFor(() => expect(page.next).toBe("active"));
    await page.advance();
    expect(calls.find((c) => c.path.endsWith("/status"))?.body).toEqual({ status: "active" });
    expect(page.match?.status).toBe("active");
    expect(page.match?.name).toBe("Spring Open");
    expect(page.next).toBe("finished");
    expect(page.error).toBe("");
  });

  it("reports a failed status change and leaves the status alone", async () => {
    const { page } = setup((call) =>
      call.path === "/api/matches/5/status" ? { status: 409, body: { error: "add entries first" } } : undefined,
    );
    page.start();
    await vi.waitFor(() => expect(page.next).toBe("active"));
    await page.advance();
    expect(page.error).toBe("add entries first");
    expect(page.match?.status).toBe("draft");
  });

  it("navigates to the admin list after deleting", async () => {
    const { page, deps } = setup((call) => (call.method === "DELETE" ? { status: 200 } : undefined));
    await page.remove();
    expect(deps.navigate).toHaveBeenCalledWith("/admin");
    expect(page.error).toBe("");
  });

  it("stays on the page and shows the error when deleting fails", async () => {
    const { page, deps } = setup((call) =>
      call.method === "DELETE" ? { status: 500, body: { error: "database busy" } } : undefined,
    );
    await page.remove();
    expect(deps.navigate).not.toHaveBeenCalled();
    expect(page.error).toBe("database busy");
  });
});
