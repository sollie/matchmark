import { api } from "./api/client";
import { settle } from "./api/util";
import { nextStatus } from "./domain";
import { navigate } from "./router.svelte";
import { openStream, type StreamHandle, type StreamOptions } from "./sse";
import type { components } from "./api/schema";

type Competition = components["schemas"]["Competition"];
type Shooter = components["schemas"]["Shooter"];
type EntryWithStages = components["schemas"]["EntryWithStages"];

export type CompetitionPageDeps = {
  openStream(options: StreamOptions): StreamHandle;
  navigate(path: string): void;
};

const browserDeps: CompetitionPageDeps = { openStream, navigate };

export class CompetitionPage {
  competition = $state<Competition | null>(null);
  shooters = $state<Shooter[]>([]);
  entries = $state<EntryWithStages[]>([]);
  error = $state("");
  version = $state(0);

  next = $derived(this.competition ? nextStatus[this.competition.status] : null);

  #id: number;
  #deps: CompetitionPageDeps;

  constructor(id: number, deps: CompetitionPageDeps = browserDeps) {
    this.#id = id;
    this.#deps = deps;
  }

  #path() {
    return { params: { path: { competitionID: this.#id } } };
  }

  loadCompetition = async () => {
    const result = await settle(api.GET("/api/competitions/{competitionID}", this.#path()));
    if (result.ok) this.competition = result.data;
    else this.error = result.error;
  };

  loadEntries = async () => {
    const result = await settle(api.GET("/api/competitions/{competitionID}/entries", this.#path()));
    if (result.ok) this.entries = result.data.entries;
    else this.error = result.error;
  };

  loadShooters = async () => {
    const result = await settle(api.GET("/api/shooters"));
    if (result.ok) this.shooters = result.data.shooters;
    else this.error = result.error;
  };

  refresh = async () => {
    await this.loadCompetition();
    await this.loadEntries();
  };

  start(): () => void {
    void this.loadCompetition();
    void this.loadShooters();
    const stream = this.#deps.openStream({
      url: `/api/competitions/${this.#id}/stream`,
      onEvent: (event) => {
        if (event === "changed") {
          void this.loadEntries();
          void this.loadCompetition();
          this.version += 1;
        }
      },
    });
    return () => stream.close();
  }

  saved(updated: Competition) {
    this.competition = updated;
  }

  async advance() {
    const current = this.competition;
    const next = this.next;
    if (!current || !next) return;
    this.error = "";
    const result = await settle(
      api.POST("/api/competitions/{competitionID}/status", { ...this.#path(), body: { status: next } }),
    );
    if (result.ok) this.competition = { ...current, status: result.data.status };
    else this.error = result.error;
  }

  async remove() {
    const result = await settle(api.DELETE("/api/competitions/{competitionID}", this.#path()));
    if (result.ok) this.#deps.navigate("/admin");
    else this.error = result.error;
  }
}
