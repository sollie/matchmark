import { api } from "./api/client";
import { openStream, type StreamHandle, type StreamOptions, type StreamState } from "./sse";
import { screenControl, type ScreenControl } from "./screenControl.svelte";
import type { components } from "./api/schema";

type Match = components["schemas"]["Match"];
type Standing = components["schemas"]["Standing"];
type Snapshot = components["schemas"]["SpectatorSnapshot"];

export type SpectatorFeedDeps = {
  openStream(options: StreamOptions): StreamHandle;
  screenControl: Pick<
    ScreenControl,
    "managed" | "assignment" | "registration" | "notice" | "error" | "start" | "stop"
  >;
};

const browserDeps: SpectatorFeedDeps = { openStream, screenControl };

export class SpectatorFeed {
  matches = $state<Match[]>([]);
  selected = $state<number | null>(null);
  standings = $state<Standing[]>([]);
  snapshot = $state<Snapshot | null>(null);
  connection = $state<StreamState>("connecting");
  updated = $state<Date | null>(null);

  #deps: SpectatorFeedDeps;

  constructor(deps: SpectatorFeedDeps = browserDeps) {
    this.#deps = deps;
  }

  get control() {
    return this.#deps.screenControl;
  }

  get viewed(): number | null {
    const control = this.#deps.screenControl;
    if (!control.managed) return this.selected;
    return control.assignment?.state === "match" ? control.assignment.match_id : null;
  }

  current = $derived(this.matches.find((c) => c.id === this.viewed));
  finished = $derived((this.current?.status ?? this.snapshot?.status) === "finished");
  acceptance = $derived.by(() => {
    const snapshot = this.snapshot;
    if (!snapshot) return null;
    const segments = snapshot.standings
      .filter((s) => !s.withdrawn)
      .flatMap((s) => s.segments?.filter((st) => st.segment === snapshot.current_segment) ?? []);
    return { accepted: segments.filter((st) => st.accepted).length, total: segments.length };
  });

  async loadMatches() {
    const { data } = await api.GET("/api/spectator/matches");
    if (!data) return;
    this.matches = data.matches;
    if (this.#deps.screenControl.managed) return;
    if (this.selected === null || !data.matches.some((c) => c.id === this.selected)) {
      this.selected =
        (data.matches.find((c) => c.status === "active") ?? data.matches[0])?.id ?? null;
    }
  }

  startScreen(): () => void {
    this.#deps.screenControl.start();
    return () => this.#deps.screenControl.stop();
  }

  watch(id: number | null): (() => void) | undefined {
    this.snapshot = null;
    this.standings = [];
    this.updated = null;
    if (id === null) return;
    const stream = this.#deps.openStream({
      url: `/api/spectator/matches/${id}/stream`,
      onEvent: (event, data) => {
        if (event !== "snapshot") return;
        const next: Snapshot = JSON.parse(data);
        this.snapshot = next;
        this.standings = next.standings ?? [];
        this.updated = new Date();
        const status = next.status;
        if (status) this.matches = this.matches.map((c) => (c.id === id ? { ...c, status } : c));
      },
      onState: (next) => {
        this.connection = next;
        if (next === "live") void this.loadMatches();
      },
    });
    return () => stream.close();
  }
}
