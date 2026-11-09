import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import ClassChips from "./ClassChips.svelte";

describe("ClassChips", () => {
  it("shows one chip per Category in Category order and hides Unclassified", () => {
    render(ClassChips, {
      classifications: {
        "Carbine 1500": { value: 4, name: "Master" },
        "Pistol Optical Sight 1500": { value: 0, name: "Unclassified" },
        "Pistol 1500": { value: 1, name: "Marksman" },
        "Revolver 1500": { value: 5, name: "High Master" },
      },
    });
    const chips = screen.getAllByRole("listitem").map((c) => c.textContent?.replace(/\s+/g, " "));
    expect(chips).toEqual([
      "Revolver 1500 High Master",
      "Pistol 1500 Marksman",
      "Carbine 1500 Master",
    ]);
  });

  it("renders nothing when no Category has a Classification", () => {
    render(ClassChips, {
      classifications: { "Revolver 1500": { value: 0, name: "Unclassified" } },
    });
    expect(screen.queryByRole("list")).toBeNull();
  });
});
