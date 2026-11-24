import { describe, expect, it, vi } from "vitest";
import { EntryEditor, fallbackCategory, nextFreeLane, isDirty, blankCounts } from "./entryEditing.svelte";
import type { EntryCommands } from "./entries";
import type { components } from "./api/schema";

type Entry = components["schemas"]["EntryWithSegments"];

const entry = (over: Partial<Entry> & { id: number; lane: number }): Entry =>
  ({
    shooter_id: over.id * 10,
    category: "Open",
    withdrawn: false,
    remarks: "",
    segments: [
      { segment_number: 1, expected_shots: 10, shots_entered: 0, status: "open", score: 0, x: 0, "10": 0, "9": 0, "8": 0, "7": 0, "0": 0, miss: 0 },
    ],
    ...over,
  }) as unknown as Entry;

function setup(entries: Entry[] = [], categories = ["Open", "Veteran"]) {
  const commands = {
    add: vi.fn(async () => null as string | null),
    update: vi.fn(async () => null as string | null),
    classify: vi.fn(async () => null as string | null),
    withdraw: vi.fn(async () => null as string | null),
    remove: vi.fn(async () => null as string | null),
    unlock: vi.fn(async () => null as string | null),
    accept: vi.fn(async () => null as string | null),
    lock: vi.fn(async () => null as string | null),
    saveCounts: vi.fn(async () => null as string | null),
  };
  const editor = new EntryEditor({
    entries: () => entries,
    categories: () => categories,
    commands: () => commands as unknown as EntryCommands,
    reload: async () => {},
  });
  return { editor, commands };
}

describe("pure rules", () => {
  it("picks the lowest lane not held by an active entry", () => {
    expect(nextFreeLane([])).toBe(1);
    expect(
      nextFreeLane([entry({ id: 1, lane: 1 }), entry({ id: 2, lane: 2, withdrawn: true }), entry({ id: 3, lane: 3 })]),
    ).toBe(2);
  });

  it("falls back to the first category when the choice is not offered", () => {
    expect(fallbackCategory(["A", "B"], "B")).toBe("B");
    expect(fallbackCategory(["A", "B"], "Z")).toBe("A");
    expect(fallbackCategory(["A", "B"], null)).toBe("A");
    expect(fallbackCategory([], null)).toBe("");
  });

  it("detects dirty counts", () => {
    const original = blankCounts();
    expect(isDirty({ ...original }, original)).toBe(false);
    expect(isDirty({ ...original, x: 2 }, original)).toBe(true);
  });
});

describe("add form", () => {
  it("defaults the lane and category and validates only after submit", async () => {
    const { editor, commands } = setup([entry({ id: 1, lane: 1 })]);
    expect(editor.lane).toBe(2);
    expect(editor.category).toBe("Open");
    expect(editor.shooterError).toBe("");
    await editor.add();
    expect(editor.shooterError).toBe("Choose a shooter.");
    expect(commands.add).not.toHaveBeenCalled();
  });

  it("rejects an invalid lane", async () => {
    const { editor, commands } = setup();
    editor.shooterId = 5;
    editor.laneInput = 0;
    await editor.add();
    expect(editor.laneError).toBe("Lane must be a whole number from 1.");
    expect(commands.add).not.toHaveBeenCalled();
  });

  it("adds and resets the form", async () => {
    const { editor, commands } = setup();
    editor.shooterId = 5;
    editor.laneInput = 3;
    await editor.add();
    expect(commands.add).toHaveBeenCalledWith({ shooter_id: 5, category: "Open", lane: 3 });
    expect(editor.shooterId).toBeNull();
    expect(editor.laneInput).toBeNull();
    expect(editor.submitted).toBe(false);
  });

  it("blocks on an archived shooter instead of showing an error", async () => {
    const { editor, commands } = setup();
    commands.add.mockResolvedValueOnce("shooter_archived");
    const shooter = { id: 5 } as components["schemas"]["Shooter"];
    editor.chosen = shooter;
    editor.shooterId = 5;
    await editor.add();
    expect(editor.error).toBe("");
    expect(editor.archivedBlock?.id).toBe(shooter.id);
  });

  it("keeps the form and shows the error on failure", async () => {
    const { editor, commands } = setup();
    commands.add.mockResolvedValueOnce("Lane taken");
    editor.shooterId = 5;
    await editor.add();
    expect(editor.error).toBe("Lane taken");
    expect(editor.shooterId).toBe(5);
  });
});

describe("dialogs", () => {
  it("reports which dialog is open", () => {
    const e = entry({ id: 1, lane: 1 });
    const { editor } = setup([e]);
    expect(editor.openDialog).toBeNull();
    editor.startEdit(e);
    expect(editor.openDialog).toBe("edit");
    editor.editing = null;
    editor.startRemarks(e);
    expect(editor.openDialog).toBe("remark");
    editor.remarking = null;
    editor.startCorrection(e);
    expect(editor.openDialog).toBe("correct");
    editor.correcting = null;
    editor.confirming = { kind: "withdraw", entry: e };
    expect(editor.openDialog).toBe("withdraw");
    editor.confirming = { kind: "remove", entry: e };
    expect(editor.openDialog).toBe("remove");
    editor.confirming = null;
    editor.openCounts(e, 1);
    expect(editor.openDialog).toBe("score");
    editor.scoring!.counts.x = 1;
    expect(editor.requestClose()).toBe(false);
    expect(editor.openDialog).toBe("discard");
  });

  it("saves an edit and closes on success only", async () => {
    const e = entry({ id: 1, lane: 1 });
    const { editor, commands } = setup([e]);
    editor.startEdit(e);
    editor.editing!.lane = 4;
    commands.update.mockResolvedValueOnce("Lane taken");
    await editor.saveEdit();
    expect(editor.editing).not.toBeNull();
    expect(editor.error).toBe("Lane taken");
    await editor.saveEdit();
    expect(commands.update).toHaveBeenLastCalledWith(1, { category: "Open", lane: 4 });
    expect(editor.editing).toBeNull();
    expect(editor.error).toBe("");
  });

  it("saves remarks with the entry's category and lane", async () => {
    const e = entry({ id: 1, lane: 2 });
    const { editor, commands } = setup([e]);
    editor.startRemarks(e);
    editor.remarking!.text = "late";
    await editor.saveRemarks(e);
    expect(commands.update).toHaveBeenCalledWith(1, { category: "Open", lane: 2, remarks: "late" });
    expect(editor.remarking).toBeNull();
  });

  it("converts class strings to numbers when correcting", async () => {
    const e = entry({ id: 1, lane: 1 });
    const { editor, commands } = setup([e]);
    editor.startCorrection(e);
    editor.correcting!.classification = "2";
    editor.correcting!.class = "3";
    await editor.saveCorrection();
    expect(commands.classify).toHaveBeenCalledWith(1, 2, 3);
    expect(editor.correcting).toBeNull();
  });

  it("closes the confirmation before running withdraw or remove", async () => {
    const e = entry({ id: 1, lane: 1 });
    const { editor, commands } = setup([e]);
    editor.confirming = { kind: "withdraw", entry: e };
    await editor.confirmed();
    expect(editor.confirming).toBeNull();
    expect(commands.withdraw).toHaveBeenCalledWith(1);
    editor.confirming = { kind: "remove", entry: e };
    await editor.confirmed();
    expect(commands.remove).toHaveBeenCalledWith(1);
  });
});

describe("score dialog", () => {
  it("closes directly when nothing changed", () => {
    const e = entry({ id: 1, lane: 1 });
    const { editor } = setup([e]);
    editor.openCounts(e, 1);
    expect(editor.requestClose()).toBe(true);
    expect(editor.scoring).toBeNull();
  });

  it("asks before discarding changes, and keep editing returns to the score dialog", () => {
    const e = entry({ id: 1, lane: 1 });
    const { editor } = setup([e]);
    editor.openCounts(e, 1);
    editor.scoring!.counts["9"] = 3;
    expect(editor.requestClose()).toBe(false);
    editor.confirmDiscard = false;
    expect(editor.openDialog).toBe("score");
    editor.requestClose();
    editor.closeCounts();
    expect(editor.openDialog).toBeNull();
  });

  it("flags stored counts that changed since opening", () => {
    const e = entry({ id: 1, lane: 1 });
    let current = [e];
    const editor = new EntryEditor({
      entries: () => current,
      categories: () => ["Open"],
      commands: () => ({}) as EntryCommands,
      reload: async () => {},
    });
    editor.openCounts(e, 1);
    expect(editor.storedChanged).toBe(false);
    const changed = entry({ id: 1, lane: 1 });
    changed.segments[0]!.x = 4;
    current = [changed];
    expect(editor.storedChanged).toBe(true);
  });

  it("keeps the dialog open and shows the error when saving fails", async () => {
    const e = entry({ id: 1, lane: 1 });
    const { editor, commands } = setup([e]);
    editor.openCounts(e, 1);
    commands.saveCounts.mockResolvedValueOnce("Counts must sum to 10");
    expect(await editor.saveCounts()).toBeNull();
    expect(editor.countsError).toBe("Counts must sum to 10");
    expect(editor.scoring).not.toBeNull();
  });

  it("closes and returns the target after saving", async () => {
    const e = entry({ id: 1, lane: 1 });
    const { editor } = setup([e]);
    editor.openCounts(e, 1);
    expect(await editor.saveCounts()).toMatchObject({ entry: { id: 1 }, segment: 1 });
    expect(editor.scoring).toBeNull();
  });
});
