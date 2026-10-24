import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/svelte";
import Scorecard from "./Scorecard.svelte";
import { stubFetch } from "../test/fetch";
import { navigate } from "../lib/router.svelte";

let originalFetch: typeof fetch;
beforeEach(() => {
  originalFetch = globalThis.fetch;
  stubFetch(() => ({ body: { name: "Club", has_logo: false, version: 5 } }));
});
afterEach(() => {
  globalThis.fetch = originalFetch;
  navigate("/");
});

describe("Scorecard", () => {
  it("shows the card image with a cache-buster and a download link", async () => {
    navigate("/scorecard/7/3");
    render(Scorecard);
    const img = (await screen.findByRole("img", { name: /Scorecard for lane 3/ })) as HTMLImageElement;
    expect(img.getAttribute("src")).toMatch(/^\/api\/scorer\/scorecard\/7\/3\?format=png&v=/);
    const link = screen.getByRole("link", { name: "Download PNG" });
    expect(link.getAttribute("href")).toBe("/api/scorer/scorecard/7/3?format=png&download=1");
  });

  it("explains when the card is not available", async () => {
    navigate("/scorecard/7/3");
    render(Scorecard);
    await fireEvent.error(await screen.findByRole("img", { name: /Scorecard for lane 3/ }));
    expect(await screen.findByText(/not available/)).toBeTruthy();
    expect(screen.queryByRole("link", { name: "Download PNG" })).toBeNull();
  });
});
