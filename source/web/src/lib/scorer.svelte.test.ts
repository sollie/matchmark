import { describe, expect, it, vi } from "vitest";
import {
  ScorerSession,
  type Outcome,
  type ScorerApi,
  type ScorerDeps,
  type Session,
} from "./scorer.svelte";
import type { StreamOptions } from "./sse";

const zero = { x: 0, 10: 0, 9: 0, 8: 0, 7: 0, 0: 0, miss: 0 };
const ok: Outcome = { ok: true, status: 200, error: "" };
const failed = (status: number, error = `Request failed (${status})`): Outcome => ({
  ok: false,
  status,
  error,
});

const snapshot = (
  overrides: Record<string, unknown> = {},
  stageOverrides: Record<string, unknown> = {},
): Session =>
  ({
    lane: 1,
    score_lane: 2,
    current_stage: 1,
    match_id: 9,
    entry_id: 4,
    shooter_name: "Rick Roe",
    phase: "scoring",
    stages: [
      {
        ...zero,
        stage_number: 1,
        expected_shots: 30,
        status: "open",
        disputed: false,
        ...stageOverrides,
      },
    ],
    ...overrides,
  }) as unknown as Session;

function setup(token: string | null = "tok") {
  const api: ScorerApi = {
    load: vi.fn(async () => ({ ...ok, session: snapshot() })),
    save: vi.fn(async () => ok),
    lock: vi.fn(async () => ok),
    respond: vi.fn(async () => ok),
  };
  const streams: { options: StreamOptions; close: ReturnType<typeof vi.fn> }[] = [];
  const deps: ScorerDeps = {
    api,
    pairing: { load: () => token, forget: vi.fn() },
    openStream: (options) => {
      const close = vi.fn();
      streams.push({ options, close });
      return { close };
    },
    navigate: vi.fn(),
  };
  return { scorer: new ScorerSession(deps), api, deps, streams };
}

describe("snapshot reconciliation", () => {
  it("resets counts and notifies when the score lane changes", () => {
    const { scorer } = setup();
    scorer.apply(snapshot());
    scorer.setCount("x", 5);
    scorer.apply(snapshot({ score_lane: 3 }));
    expect(scorer.counts.x).toBe(0);
    expect(scorer.notice).toBe(
      "Score lane changed to lane 3. Your unsaved ring counts were reset.",
    );
  });

  it("keeps unsaved counts across snapshots for the same stage", () => {
    const { scorer } = setup();
    scorer.apply(snapshot());
    scorer.setCount("x", 5);
    scorer.apply(snapshot());
    expect(scorer.counts.x).toBe(5);
    expect(scorer.unsaved).toBe(true);
  });

  it("replaces counts when the stage is newly disputed", () => {
    const { scorer } = setup();
    scorer.apply(snapshot());
    scorer.setCount("x", 5);
    scorer.apply(snapshot({}, { disputed: true, x: 30 }));
    expect(scorer.counts.x).toBe(30);
    expect(scorer.canLock).toBe(true);
  });

  it("forgets the device and navigates to the scorecard when finished", () => {
    const { scorer, deps } = setup();
    scorer.apply(snapshot({ phase: "finished" }));
    expect(deps.pairing.forget).toHaveBeenCalled();
    expect(deps.navigate).toHaveBeenCalledWith("/scorecard/9/1");
    expect(scorer.token).toBeNull();
    expect(scorer.session).toBeNull();
  });
});

describe("validation", () => {
  it("caps counts at the expected shots and reports a full stage", () => {
    const { scorer } = setup();
    scorer.apply(snapshot());
    expect(scorer.setCount("x", 20)).toBe(false);
    expect(scorer.setCount("10", 15)).toBe(true);
    expect(scorer.counts["10"]).toBe(10);
    expect(scorer.stageFull).toBe(true);
    expect(scorer.atCap).toBe(true);
  });

  it("clamps negative and non-numeric input to zero", () => {
    const { scorer } = setup();
    scorer.apply(snapshot());
    scorer.setCount("x", -4);
    scorer.setCount("9", Number.NaN);
    expect(scorer.counts.x).toBe(0);
    expect(scorer.counts["9"]).toBe(0);
  });

  it("explains a 422 on save and lock", async () => {
    const { scorer, api } = setup();
    scorer.apply(snapshot());
    scorer.setCount("x", 10);
    vi.mocked(api.save).mockResolvedValueOnce(failed(422));
    await scorer.save();
    expect(scorer.error).toBe("Ring counts must add up to 30 shots (entered 10).");
    vi.mocked(api.lock).mockResolvedValueOnce(failed(422));
    await scorer.lock();
    expect(scorer.error).toBe("Save valid ring counts before locking the stage.");
  });
});

describe("save, lock and respond", () => {
  it("marks counts saved and locks the stage", async () => {
    const { scorer, api } = setup();
    scorer.apply(snapshot());
    scorer.setCount("x", 30);
    await scorer.save();
    expect(api.save).toHaveBeenCalledWith("tok", 1, expect.objectContaining({ x: 30 }));
    expect(scorer.unsaved).toBe(false);
    await scorer.lock();
    expect(api.lock).toHaveBeenCalledWith("tok", 1);
    expect(scorer.message).toBe("Stage locked");
    expect(scorer.busy).toBe(false);
  });

  it("accepts a result", async () => {
    const { scorer, api } = setup();
    scorer.apply(
      snapshot({
        phase: "needs_acceptance",
        acceptance: { ...zero, stage_number: 1, total: 0, scorer_name: "A" },
      }),
    );
    scorer.startReview();
    await scorer.respond("accept");
    expect(api.respond).toHaveBeenCalledWith("tok", 1, "accept");
    expect(scorer.reviewing).toBe(false);
    expect(scorer.message).toBe("Result accepted");
  });

  it("keeps reviewing and shows the error when a dispute fails", async () => {
    const { scorer, api } = setup();
    scorer.apply(
      snapshot({
        phase: "needs_acceptance",
        acceptance: { ...zero, stage_number: 1, total: 0, scorer_name: "A" },
      }),
    );
    scorer.startReview();
    scorer.askDispute();
    vi.mocked(api.respond).mockResolvedValueOnce(failed(409, "Stage already accepted"));
    await scorer.respond("dispute");
    expect(scorer.error).toBe("Stage already accepted");
    expect(scorer.reviewing).toBe(true);
    expect(scorer.confirmingDispute).toBe(false);
  });
});

describe("device token and live updates", () => {
  it("forgets the device when the token is expired", async () => {
    const { scorer, api, deps } = setup();
    vi.mocked(api.load).mockResolvedValueOnce(failed(410));
    await scorer.load();
    expect(deps.pairing.forget).toHaveBeenCalled();
    expect(scorer.token).toBeNull();
  });

  it("shows the load error for other failures", async () => {
    const { scorer, api } = setup();
    vi.mocked(api.load).mockResolvedValueOnce(failed(500));
    await scorer.load();
    expect(scorer.loadError).toBe("Request failed (500)");
    expect(scorer.token).toBe("tok");
  });

  it("does not open a stream without a token", () => {
    const { scorer, streams } = setup(null);
    expect(scorer.start()).toBeUndefined();
    expect(streams).toHaveLength(0);
  });

  it("applies snapshots, handles revocation and reloads when the stream ends", async () => {
    const { scorer, api, streams } = setup();
    const stop = scorer.start()!;
    const { options } = streams[0];
    options.onEvent("snapshot", JSON.stringify(snapshot()));
    expect(scorer.session?.shooter_name).toBe("Rick Roe");
    options.onState?.("stale");
    expect(scorer.connection).toBe("stale");
    options.onState?.("ended");
    await vi.waitFor(() => expect(api.load).toHaveBeenCalledWith("tok"));
    options.onEvent("revoked", "");
    expect(scorer.token).toBeNull();
    stop();
    expect(streams[0].close).toHaveBeenCalled();
  });

  it("does not reload on end after being disposed", () => {
    const { scorer, api, streams } = setup();
    const stop = scorer.start()!;
    stop();
    streams[0].options.onState?.("ended");
    expect(api.load).not.toHaveBeenCalled();
  });
});
