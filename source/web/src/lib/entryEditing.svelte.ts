import { api } from "./api/client";
import { settle } from "./api/util";
import { Command } from "./command.svelte";
import type { EntryCommands } from "./entries";
import type { components } from "./api/schema";

type Shooter = components["schemas"]["Shooter"];
type Entry = components["schemas"]["EntryWithSegments"];
type RingCounts = components["schemas"]["RingCounts"];

export const ringKeys = ["x", "10", "9", "8", "7", "0", "miss"] as const;

export type OpenDialog =
  | "add"
  | "edit"
  | "remark"
  | "correct"
  | "withdraw"
  | "remove"
  | "score"
  | "discard";

export const blankCounts = (): RingCounts => ({
  x: 0,
  "10": 0,
  "9": 0,
  "8": 0,
  "7": 0,
  "0": 0,
  miss: 0,
});

export function nextFreeLane(entries: Entry[]): number {
  const taken = new Set(entries.filter((e) => !e.withdrawn).map((e) => e.lane));
  let n = 1;
  while (taken.has(n)) n++;
  return n;
}

export function fallbackCategory(categories: string[], chosen: string | null): string {
  return chosen !== null && categories.includes(chosen) ? chosen : (categories[0] ?? "");
}

export function isDirty(counts: RingCounts, original: RingCounts): boolean {
  return ringKeys.some((k) => (Number(counts[k]) || 0) !== original[k]);
}

export const storedKey = (s: { [k in (typeof ringKeys)[number]]: number } | undefined): string =>
  ringKeys.map((k) => s?.[k] ?? 0).join(",");

type Deps = {
  entries: () => Entry[];
  categories: () => string[];
  commands: () => EntryCommands;
  reload: () => Promise<void>;
};

export class EntryEditor {
  shooterId = $state<number | null>(null);
  chosen = $state<Shooter | null>(null);
  chosenCategory = $state<string | null>(null);
  laneInput = $state<number | null>(null);
  submitted = $state(false);
  archivedBlock = $state<Shooter | null>(null);
  editing = $state<{ id: number; category: string; lane: number } | null>(null);
  remarking = $state<{ id: number; text: string } | null>(null);
  correcting = $state<{ id: number; classification: string; class: string } | null>(null);
  confirming = $state<{ kind: "withdraw" | "remove"; entry: Entry } | null>(null);
  scoring = $state<{
    entry: Entry;
    segment: number;
    counts: RingCounts;
    original: RingCounts;
  } | null>(null);
  confirmDiscard = $state(false);
  countsError = $state("");
  openedStored = $state("");

  readonly #deps: Deps;
  readonly #command: Command;

  constructor(deps: Deps) {
    this.#deps = deps;
    this.#command = new Command(deps.reload);
  }

  get error(): string {
    return this.#command.error;
  }

  get nextLane(): number {
    return nextFreeLane(this.#deps.entries());
  }

  get lane(): number {
    return this.laneInput ?? this.nextLane;
  }

  get category(): string {
    return fallbackCategory(this.#deps.categories(), this.chosenCategory);
  }

  get shooterError(): string {
    return this.submitted && this.shooterId === null ? "Choose a shooter." : "";
  }

  get laneError(): string {
    const lane = this.lane;
    return this.submitted && !(Number.isInteger(Number(lane)) && Number(lane) >= 1)
      ? "Lane must be a whole number from 1."
      : "";
  }

  get openDialog(): OpenDialog | null {
    if (this.confirmDiscard) return "discard";
    if (this.scoring) return "score";
    if (this.confirming) return this.confirming.kind;
    if (this.correcting) return "correct";
    if (this.remarking) return "remark";
    if (this.editing) return "edit";
    return null;
  }

  get storedChanged(): boolean {
    if (!this.scoring) return false;
    const { entry, segment } = this.scoring;
    return storedKey(this.storedSegment(entry.id, segment)) !== this.openedStored;
  }

  storedSegment(entryId: number, segment: number) {
    return this.#deps
      .entries()
      .find((e) => e.id === entryId)
      ?.segments.find((s) => s.segment_number === segment);
  }

  clearError(): void {
    this.#command.clear();
  }

  setError(message: string): void {
    this.#command.fail(message);
  }

  async #run(command: Promise<string | null>): Promise<boolean> {
    this.#command.clear();
    const failed = await command;
    if (failed !== null) this.#command.fail(failed);
    return failed === null;
  }

  selectShooter(shooter: Shooter | null): void {
    this.shooterId = shooter?.id ?? null;
    this.chosen = shooter;
    this.archivedBlock = null;
  }

  async add(): Promise<void> {
    this.submitted = true;
    if (this.shooterId === null || this.laneError) return;
    this.archivedBlock = null;
    const ok = await this.#run(
      this.#deps
        .commands()
        .add({ shooter_id: this.shooterId, category: this.category, lane: this.lane }),
    );
    if (!ok && this.error === "shooter_archived") {
      this.#command.clear();
      this.archivedBlock = this.chosen;
    }
    if (ok) {
      this.laneInput = null;
      this.shooterId = null;
      this.chosen = null;
      this.submitted = false;
    }
  }

  async unarchive(): Promise<void> {
    if (!this.archivedBlock) return;
    const result = await settle(
      api.POST("/api/shooters/{shooterID}/unarchive", {
        params: { path: { shooterID: this.archivedBlock.id } },
      }),
    );
    if (result.ok) this.archivedBlock = null;
    else this.#command.fail(result.error);
  }

  startEdit(entry: Entry): void {
    this.editing = { id: entry.id, category: entry.category, lane: entry.lane };
  }

  async saveEdit(): Promise<void> {
    if (!this.editing) return;
    const ok = await this.#run(
      this.#deps
        .commands()
        .update(this.editing.id, { category: this.editing.category, lane: this.editing.lane }),
    );
    if (ok) this.editing = null;
  }

  startRemarks(entry: Entry): void {
    this.remarking = { id: entry.id, text: entry.remarks ?? "" };
  }

  async saveRemarks(entry: Entry): Promise<void> {
    if (!this.remarking) return;
    const ok = await this.#run(
      this.#deps.commands().update(entry.id, {
        category: entry.category,
        lane: entry.lane,
        remarks: this.remarking.text,
      }),
    );
    if (ok) this.remarking = null;
  }

  startCorrection(entry: Entry): void {
    this.correcting = {
      id: entry.id,
      classification: String(entry.classification_at_match?.value ?? 0),
      class: String(entry.class?.value ?? 0),
    };
  }

  async saveCorrection(): Promise<void> {
    if (!this.correcting) return;
    const { id, classification, class: competed } = this.correcting;
    const ok = await this.#run(
      this.#deps.commands().classify(id, Number(classification), Number(competed)),
    );
    if (ok) this.correcting = null;
  }

  async confirmed(): Promise<void> {
    if (!this.confirming) return;
    const { kind, entry } = this.confirming;
    this.confirming = null;
    const commands = this.#deps.commands();
    await this.#run(kind === "withdraw" ? commands.withdraw(entry.id) : commands.remove(entry.id));
  }

  unlock(entry: Entry, segment: number): Promise<boolean> {
    return this.#run(this.#deps.commands().unlock(entry.id, segment));
  }

  accept(entry: Entry, segment: number): Promise<boolean> {
    return this.#run(this.#deps.commands().accept(entry.id, segment));
  }

  lock(entry: Entry, segment: number): Promise<boolean> {
    return this.#run(this.#deps.commands().lock(entry.id, segment));
  }

  openCounts(entry: Entry, segment: number): void {
    this.countsError = "";
    this.#command.clear();
    this.confirmDiscard = false;
    const stored = entry.segments.find((s) => s.segment_number === segment);
    const counts = blankCounts();
    for (const k of ringKeys) counts[k] = stored?.[k] ?? 0;
    this.scoring = { entry, segment, counts, original: { ...counts } };
    this.openedStored = storedKey(stored);
  }

  closeCounts(): void {
    this.scoring = null;
    this.countsError = "";
    this.confirmDiscard = false;
  }

  requestClose(): boolean {
    if (!this.scoring) return true;
    if (isDirty(this.scoring.counts, this.scoring.original)) {
      this.confirmDiscard = true;
      return false;
    }
    this.closeCounts();
    return true;
  }

  async saveCounts(): Promise<{ entry: Entry; segment: number } | null> {
    if (!this.scoring) return null;
    const { entry, segment, counts } = this.scoring;
    this.countsError = (await this.#deps.commands().saveCounts(entry.id, segment, counts)) ?? "";
    if (this.countsError) return null;
    this.closeCounts();
    return { entry, segment };
  }
}
