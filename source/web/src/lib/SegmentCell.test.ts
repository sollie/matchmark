import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import SegmentCell from "./SegmentCell.svelte";

describe("SegmentCell", () => {
  it.each([
    ["not_started", undefined, "", "Segment 3: not started"],
    ["in_progress", undefined, "", "Segment 3: in progress"],
    ["awaiting_acceptance", undefined, "", "Segment 3: awaiting acceptance"],
    ["accepted", 118, "118", "Segment 3: accepted, 118"],
    ["accepted", undefined, "", "Segment 3: accepted"],
  ] as const)("renders %s with score %s as %s", (state, score, text, label) => {
    render(SegmentCell, { segment: { segment: 3, accepted: state === "accepted", state, score } });
    const cell = screen.getByLabelText(label);
    expect(cell.textContent?.trim()).toBe(text);
    expect(cell.getAttribute("title")).toBe(label);
    expect(cell.className).toContain(state);
  });

  it("omits the segment number when used as a legend key", () => {
    render(SegmentCell, {
      segment: { segment: 0, accepted: false, state: "in_progress" },
      prefix: false,
    });
    expect(screen.getByLabelText("in progress").textContent?.trim()).toBe("");
  });

  it("renders nothing meaningful for a missing segment", () => {
    render(SegmentCell, { segment: undefined });
    expect(screen.queryByLabelText(/Segment/)).toBeNull();
  });
});
