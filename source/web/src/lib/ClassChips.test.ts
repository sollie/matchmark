import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import ClassChips from "./ClassChips.svelte";

describe("ClassChips", () => {
  it("shows one chip per Category in Category order and hides Unclassified", () => {
    render(ClassChips, {
      classifications: {
        Carbine: { value: 4, name: "Master" },
        "Pistol OS": { value: 0, name: "Unclassified" },
        Pistol: { value: 1, name: "Marksman" },
        Revolver: { value: 5, name: "High Master" },
      },
    });
    const chips = screen.getAllByRole("listitem").map((c) => c.textContent?.replace(/\s+/g, " "));
    expect(chips).toEqual(["Revolver High Master", "Pistol Marksman", "Carbine Master"]);
  });

  it("renders nothing when no Category has a Classification", () => {
    render(ClassChips, { classifications: { Revolver: { value: 0, name: "Unclassified" } } });
    expect(screen.queryByRole("list")).toBeNull();
  });
});
