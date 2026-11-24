import { describe, expect, it, vi } from "vitest";
import { SpectatorFeed, type SpectatorFeedDeps } from "./spectatorFeed.svelte";
import { ScreenControl } from "./screenControl.svelte";
import { stubFetch, type Call } from "../test/fetch";
import type { StreamOptions } from "./sse";

const match = (id: number, status = "active") => ({
  id,
  name: `Match ${id}`,
  date: "2026-05-01",
  course: "150",
  status,
});

const standing = (lane: number) => ({ lane, category: "Open", withdrawn: false, total: 0 });

function setup(matches = [match(7)]) {
  const calls = stubFetch((call: Call) =>
    call.path === "/api/spectator/matches" ? { body: { matches } } : undefined,
  );
  const streams: { options: StreamOptions; close: ReturnType<typeof vi.fn> }[] = [];
  const screen = new ScreenControl();
  screen.registration = null;
  const deps: SpectatorFeedDeps = {
    openStream: (options) => {
      const close = vi.fn();
      streams.push({ options, close });
      return { close };
    },
    screenControl: screen,
  };
  const feed = new SpectatorFeed(deps);
  const lastStream = () => streams[streams.length - 1];
  const emit = (data: unknown) => lastStream().options.onEvent("snapshot", JSON.stringify(data));
  const loads = () => calls.filter((c) => c.path === "/api/spectator/matches").length;
  return { feed, screen, streams, calls, emit, lastStream, loads };
}

describe("match list", () => {
  it("selects the first active match", async () => {
    const { feed } = setup([match(1, "finished"), match(2, "active"), match(3, "active")]);
    await feed.loadMatches();
    expect(feed.selected).toBe(2);
    expect(feed.viewed).toBe(2);
  });

  it("falls back to the first match when none is active", async () => {
    const { feed } = setup([match(1, "finished"), match(2, "finished")]);
    await feed.loadMatches();
    expect(feed.selected).toBe(1);
  });

  it("keeps a still-listed selection and replaces a vanished one", async () => {
    const { feed } = setup([match(1), match(2)]);
    await feed.loadMatches();
    feed.selected = 2;
    await feed.loadMatches();
    expect(feed.selected).toBe(2);
    feed.selected = 99;
    await feed.loadMatches();
    expect(feed.selected).toBe(1);
  });
});

describe("snapshot stream", () => {
  it("opens a stream for the viewed match and exposes snapshots", async () => {
    const { feed, streams, emit } = setup();
    await feed.loadMatches();
    feed.watch(feed.viewed);
    expect(streams[0].options.url).toBe("/api/spectator/matches/7/stream");
    emit({ current_segment: 2, total_segments: 10, standings: [standing(1)] });
    expect(feed.snapshot?.current_segment).toBe(2);
    expect(feed.standings).toHaveLength(1);
    expect(feed.updated).toBeInstanceOf(Date);
  });

  it("ignores events other than snapshot", async () => {
    const { feed, lastStream } = setup();
    await feed.loadMatches();
    feed.watch(feed.viewed);
    lastStream().options.onEvent("ping", "{}");
    expect(feed.snapshot).toBeNull();
  });

  it("opens no stream without a viewed match and clears state", () => {
    const { feed, streams } = setup();
    feed.watch(null);
    expect(streams).toHaveLength(0);
    expect(feed.snapshot).toBeNull();
  });

  it("closes the stream and clears state when stopped or switching", async () => {
    const { feed, streams, emit } = setup();
    await feed.loadMatches();
    const stop = feed.watch(7);
    emit({ current_segment: 1, total_segments: 2, standings: [standing(1)] });
    stop?.();
    expect(streams[0].close).toHaveBeenCalled();
    feed.watch(7);
    expect(feed.snapshot).toBeNull();
    expect(feed.standings).toEqual([]);
    expect(feed.updated).toBeNull();
  });

  it("tracks the connection state", async () => {
    const { feed, lastStream } = setup();
    await feed.loadMatches();
    feed.watch(7);
    lastStream().options.onState?.("stale");
    expect(feed.connection).toBe("stale");
  });

  it("reloads the match list when the stream goes live", async () => {
    const { feed, lastStream, loads } = setup();
    await feed.loadMatches();
    feed.watch(7);
    const before = loads();
    lastStream().options.onState?.("live");
    await vi.waitFor(() => expect(loads()).toBe(before + 1));
  });

  it("notices a match finishing from a snapshot without reloading the list", async () => {
    const { feed, emit, loads } = setup();
    await feed.loadMatches();
    feed.watch(7);
    const before = loads();
    emit({ status: "finished", current_segment: 10, total_segments: 10, standings: [] });
    expect(feed.current?.status).toBe("finished");
    expect(feed.finished).toBe(true);
    expect(loads()).toBe(before);
  });
});

describe("acceptance", () => {
  it("counts accepted segments of the current segment, skipping withdrawn", async () => {
    const { feed, emit } = setup();
    await feed.loadMatches();
    feed.watch(7);
    const lane = (n: number, state: string, withdrawn = false) => ({
      ...standing(n),
      withdrawn,
      segments: [{ segment: 2, accepted: state === "accepted", state }],
    });
    emit({
      current_segment: 2,
      total_segments: 10,
      standings: [lane(1, "accepted"), lane(2, "in_progress"), lane(3, "accepted", true)],
    });
    expect(feed.acceptance).toEqual({ accepted: 1, total: 2 });
  });

  it("is null before a snapshot arrives", () => {
    expect(setup().feed.acceptance).toBeNull();
  });
});

describe("managed screen", () => {
  it("views the assigned match instead of the selection", async () => {
    const { feed, screen } = setup([match(7), match(8)]);
    await feed.loadMatches();
    expect(feed.viewed).toBe(7);
    screen.registration = { token: "tok", code: "ABC234" };
    expect(feed.viewed).toBeNull();
    screen.assignment = { state: "match", match_id: 8, approved: true };
    expect(feed.viewed).toBe(8);
    screen.assignment = { state: "waiting", match_id: null, approved: true };
    expect(feed.viewed).toBeNull();
  });

  it("leaves the selection alone while managed", async () => {
    const { feed, screen } = setup([match(7)]);
    screen.registration = { token: "tok", code: "ABC234" };
    await feed.loadMatches();
    expect(feed.matches).toHaveLength(1);
    expect(feed.selected).toBeNull();
  });

  it("starts and stops screen control with the returned cleanup", () => {
    const { feed, screen } = setup();
    const start = vi.spyOn(screen, "start");
    const stop = vi.spyOn(screen, "stop");
    const cleanup = feed.startScreen();
    expect(start).toHaveBeenCalled();
    cleanup();
    expect(stop).toHaveBeenCalled();
  });
});
