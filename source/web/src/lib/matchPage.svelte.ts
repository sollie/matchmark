import { api } from "./api/client";
import { settle } from "./api/util";
import { nextStatus } from "./domain";
import { navigate } from "./router.svelte";
import { openStream, type StreamHandle, type StreamOptions } from "./sse";
import type { components } from "./api/schema";

type Match = components["schemas"]["Match"];
type Shooter = components["schemas"]["Shooter"];
type EntryWithStages = components["schemas"]["EntryWithStages"];

export type MatchPageDeps = {
  openStream(options: StreamOptions): StreamHandle;
  navigate(path: string): void;
};

const browserDeps: MatchPageDeps = { openStream, navigate };

export class MatchPage {
  match = $state<Match | null>(null);
  shooters = $state<Shooter[]>([]);
  entries = $state<EntryWithStages[]>([]);
  error = $state("");
  version = $state(0);

  next = $derived(this.match ? nextStatus[this.match.status] : null);

  #id: number;
  #deps: MatchPageDeps;

  constructor(id: number, deps: MatchPageDeps = browserDeps) {
    this.#id = id;
    this.#deps = deps;
  }

  #path() {
    return { params: { path: { matchID: this.#id } } };
  }

  loadMatch = async () => {
    const result = await settle(api.GET("/api/matches/{matchID}", this.#path()));
    if (result.ok) this.match = result.data;
    else this.error = result.error;
  };

  loadEntries = async () => {
    const result = await settle(api.GET("/api/matches/{matchID}/entries", this.#path()));
    if (result.ok) this.entries = result.data.entries;
    else this.error = result.error;
  };

  loadShooters = async () => {
    const result = await settle(api.GET("/api/shooters"));
    if (result.ok) this.shooters = result.data.shooters;
    else this.error = result.error;
  };

  refresh = async () => {
    await this.loadMatch();
    await this.loadEntries();
  };

  start(): () => void {
    void this.loadMatch();
    void this.loadShooters();
    const stream = this.#deps.openStream({
      url: `/api/matches/${this.#id}/stream`,
      onEvent: (event) => {
        if (event === "changed") {
          void this.loadEntries();
          void this.loadMatch();
          this.version += 1;
        }
      },
    });
    return () => stream.close();
  }

  saved(updated: Match) {
    this.match = updated;
  }

  async advance() {
    const current = this.match;
    const next = this.next;
    if (!current || !next) return;
    this.error = "";
    const result = await settle(
      api.POST("/api/matches/{matchID}/status", { ...this.#path(), body: { status: next } }),
    );
    if (result.ok) this.match = { ...current, status: result.data.status };
    else this.error = result.error;
  }

  async remove() {
    const result = await settle(api.DELETE("/api/matches/{matchID}", this.#path()));
    if (result.ok) this.#deps.navigate("/admin");
    else this.error = result.error;
  }
}
