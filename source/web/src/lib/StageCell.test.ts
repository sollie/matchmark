import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import StageCell from "./StageCell.svelte";

describe("StageCell", () => {
  it.each([
    ["not_started", undefined, "", "Stage 3: not started"],
    ["in_progress", undefined, "", "Stage 3: in progress"],
    ["awaiting_acceptance", undefined, "", "Stage 3: awaiting acceptance"],
    ["accepted", 118, "118", "Stage 3: accepted, 118"],
    ["accepted", undefined, "", "Stage 3: accepted"],
  ] as const)("renders %s with score %s as %s", (state, score, text, label) => {
    render(StageCell, { stage: { stage: 3, accepted: state === "accepted", state, score } });
    const cell = screen.getByLabelText(label);
    expect(cell.textContent?.trim()).toBe(text);
    expect(cell.getAttribute("title")).toBe(label);
    expect(cell.className).toContain(state);
  });

  it("omits the stage number when used as a legend key", () => {
    render(StageCell, { stage: { stage: 0, accepted: false, state: "in_progress" }, prefix: false });
    expect(screen.getByLabelText("in progress").textContent?.trim()).toBe("");
  });

  it("renders nothing meaningful for a missing stage", () => {
    render(StageCell, { stage: undefined });
    expect(screen.queryByLabelText(/Stage/)).toBeNull();
  });
});
