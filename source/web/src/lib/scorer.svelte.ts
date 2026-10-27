import { api, deviceHeaders } from "./api/client";
import { errorMessage } from "./api/util";
import { openStream, type StreamHandle, type StreamOptions, type StreamState } from "./sse";
import { navigate } from "./router.svelte";
import { forgetDeviceToken, loadDeviceToken } from "./pairing";
import type { components } from "./api/schema";

export type Session = components["schemas"]["ScorerSession"];
type Rings = components["schemas"]["RingCounts"];
type SessionStage = Session["stages"][number];

export const rings = [
  { key: "x", label: "X" },
  { key: "10", label: "10" },
  { key: "9", label: "9" },
  { key: "8", label: "8" },
  { key: "7", label: "7" },
  { key: "0", label: "0" },
  { key: "miss", label: "Miss" },
] as const;

export type Outcome = { ok: boolean; status: number; error: string };
export type LoadOutcome = Outcome & { session?: Session };

export type ScorerApi = {
  load(token: string): Promise<LoadOutcome>;
  save(token: string, stage: number, counts: Record<string, number>): Promise<Outcome>;
  lock(token: string, stage: number): Promise<Outcome>;
  respond(token: string, stage: number, action: "accept" | "dispute"): Promise<Outcome>;
};

export type ScorerDeps = {
  api: ScorerApi;
  pairing: { load(): string | null; forget(): void };
  openStream(options: StreamOptions): StreamHandle;
  navigate(path: string): void;
};

const emptyCounts = (): Record<string, number> => Object.fromEntries(rings.map((r) => [r.key, 0]));

const savedCounts = (s: SessionStage | undefined): Record<string, number> =>
  s ? Object.fromEntries(rings.map((r) => [r.key, s[r.key]])) : emptyCounts();

function outcome(result: { error?: unknown; response: Response }): Outcome {
  return {
    ok: result.response.ok,
    status: result.response.status,
    error: errorMessage(result.error, `Request failed (${result.response.status})`),
  };
}

export const httpScorerApi: ScorerApi = {
  async load(token) {
    const result = await api.GET("/api/scorer/session", { headers: deviceHeaders(token) });
    return { ...outcome(result), session: result.data };
  },
  async save(token, stage, counts) {
    return outcome(
      await api.PUT("/api/scorer/stages/{stage}", {
        params: { path: { stage } },
        headers: deviceHeaders(token),
        body: {
          x: counts.x,
          10: counts["10"],
          9: counts["9"],
          8: counts["8"],
          7: counts["7"],
          0: counts["0"],
          miss: counts.miss,
        } as Rings,
      }),
    );
  },
  async lock(token, stage) {
    return outcome(
      await api.POST("/api/scorer/stages/{stage}/lock", {
        params: { path: { stage } },
        headers: deviceHeaders(token),
      }),
    );
  },
  async respond(token, stage, action) {
    return outcome(
      await api.POST(
        action === "accept" ? "/api/scorer/stages/{stage}/accept" : "/api/scorer/stages/{stage}/dispute",
        { params: { path: { stage } }, headers: deviceHeaders(token) },
      ),
    );
  },
};

export const browserScorerDeps: ScorerDeps = {
  api: httpScorerApi,
  pairing: { load: loadDeviceToken, forget: forgetDeviceToken },
  openStream,
  navigate,
};

export class ScorerSession {
  token = $state<string | null>(null);
  session = $state<Session | null>(null);
  loadError = $state("");
  counts = $state<Record<string, number>>(emptyCounts());
  saved = $state<Record<string, number>>(emptyCounts());
  message = $state("");
  error = $state("");
  busy = $state(false);
  notice = $state("");
  connection = $state<StreamState>("connecting");
  connectAttempt = $state(0);
  reviewing = $state(false);
  started = $state(false);
  confirmingDispute = $state(false);
  stageFull = $state(false);

  stage = $derived(this.session?.current_stage ?? null);
  current = $derived(this.session?.stages.find((s) => s.stage_number === this.stage) ?? null);
  waitingStage = $derived(
    this.session?.stages.find((s) => s.stage_number === this.stage && s.status !== "accepted") ?? null,
  );
  acceptance = $derived(this.session?.acceptance ?? null);
  maxRing = $derived.by(() => {
    const acceptance = this.acceptance;
    return acceptance ? Math.max(1, ...rings.map((r) => acceptance[r.key])) : 1;
  });
  announcement = $derived.by(() => {
    const session = this.session;
    if (!session) return "";
    switch (session.phase) {
      case "not_started":
        return "Status: waiting for the match to start.";
      case "next_stage":
        return `Status: stage ${session.current_stage} started.`;
      case "needs_acceptance":
        return "Status: action needed, review your result.";
      case "waiting":
        return "Status: waiting.";
      default:
        return `Status: scoring lane ${session.score_lane}.`;
    }
  });
  sum = $derived(rings.reduce((n, r) => n + this.counts[r.key], 0));
  atCap = $derived(this.current !== null && this.sum >= this.current.expected_shots);
  unsaved = $derived(rings.some((r) => this.counts[r.key] !== this.saved[r.key]));
  savedValid = $derived(
    this.current !== null && rings.reduce((n, r) => n + this.saved[r.key], 0) === this.current.expected_shots,
  );
  canLock = $derived(!this.unsaved && this.savedValid);
  saveStatus = $derived(
    this.unsaved
      ? "Unsaved changes: press Save, then Lock stage."
      : this.savedValid
        ? "Saved ✓"
        : `Enter ring counts that add up to ${this.current?.expected_shots} shots, then press Save.`,
  );

  #deps: ScorerDeps;

  constructor(deps: ScorerDeps = browserScorerDeps) {
    this.#deps = deps;
    this.token = deps.pairing.load();
  }

  async load() {
    if (!this.token) return;
    const result = await this.#deps.api.load(this.token);
    if (result.session) {
      this.apply(result.session);
      this.loadError = "";
    } else if (result.status === 401 || result.status === 410) {
      this.forget();
    } else {
      this.loadError = result.error;
    }
  }

  forget() {
    this.#deps.pairing.forget();
    this.token = null;
    this.session = null;
  }

  unpair() {
    this.forget();
  }

  reconnect() {
    this.loadError = "";
    this.connectAttempt++;
  }

  start(): (() => void) | undefined {
    if (!this.token) return;
    let disposed = false;
    const stream = this.#deps.openStream({
      url: "/api/scorer/stream",
      headers: deviceHeaders(this.token),
      onEvent: (event, data) => {
        if (event === "snapshot") {
          this.loadError = "";
          this.apply(JSON.parse(data));
        } else if (event === "revoked") {
          this.forget();
        }
      },
      onState: (next) => {
        this.connection = next;
        if (next === "ended" && !disposed) void this.load();
      },
    });
    return () => {
      disposed = true;
      stream.close();
    };
  }

  apply(next: Session) {
    if (next.phase === "finished") {
      this.forget();
      this.#deps.navigate(`/scorecard/${next.match_id}/${next.lane}`);
      return;
    }
    const previous = this.session;
    this.session = next;
    if (previous && previous.score_lane !== next.score_lane) {
      this.counts = emptyCounts();
      this.message = "";
      this.error = "";
      this.notice = `Score lane changed to lane ${next.score_lane}. Your unsaved ring counts were reset.`;
    }
    const now = next.stages.find((s) => s.stage_number === next.current_stage);
    this.saved = savedCounts(now);
    if (!previous || previous.current_stage !== next.current_stage) {
      this.counts = savedCounts(now);
      this.message = "";
      this.error = "";
    } else {
      const was = previous.stages.find((s) => s.stage_number === next.current_stage);
      if (now?.disputed && !was?.disputed) this.counts = savedCounts(now);
    }
    if (next.phase !== "next_stage") this.started = false;
    if (next.phase !== "needs_acceptance") {
      this.reviewing = false;
      this.confirmingDispute = false;
    }
  }

  setCount(key: string, value: number): boolean {
    const wanted = Number.isFinite(value) ? Math.max(0, Math.floor(value)) : 0;
    const room = Math.max(0, (this.current?.expected_shots ?? Infinity) - this.sum);
    const next = wanted > this.counts[key] ? Math.min(wanted, this.counts[key] + room) : wanted;
    this.stageFull = next < wanted;
    this.counts[key] = next;
    return next !== wanted;
  }

  bump(key: string, delta: number) {
    this.setCount(key, this.counts[key] + delta);
  }

  continueToScoring() {
    this.started = true;
  }

  startReview() {
    this.reviewing = true;
  }

  askDispute() {
    this.confirmingDispute = true;
  }

  cancelDispute() {
    this.confirmingDispute = false;
  }

  async save() {
    const stage = this.stage;
    if (stage === null || !this.token) return;
    this.#begin();
    this.notice = "";
    this.message = "";
    const result = await this.#deps.api.save(this.token, stage, this.counts);
    this.busy = false;
    if (result.ok) this.saved = { ...this.counts };
    else if (result.status === 422)
      this.error = `Ring counts must add up to ${this.current?.expected_shots} shots (entered ${this.sum}).`;
    else this.error = result.error;
  }

  async lock() {
    const stage = this.stage;
    if (stage === null || !this.token) return;
    this.#begin();
    this.notice = "";
    this.message = "";
    const result = await this.#deps.api.lock(this.token, stage);
    this.busy = false;
    if (result.ok) this.message = "Stage locked";
    else if (result.status === 422) this.error = "Save valid ring counts before locking the stage.";
    else this.error = result.error;
  }

  async respond(action: "accept" | "dispute") {
    const acceptance = this.acceptance;
    if (!acceptance || !this.token) return;
    this.#begin();
    const result = await this.#deps.api.respond(this.token, acceptance.stage_number, action);
    this.busy = false;
    this.confirmingDispute = false;
    if (result.ok) {
      this.reviewing = false;
      this.message = action === "accept" ? "Result accepted" : "Result disputed";
    } else {
      this.error = result.error;
    }
  }

  #begin() {
    this.busy = true;
    this.error = "";
  }
}
