import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import Scorecard from "./Scorecard.svelte";
import { stubFetch, type Call } from "../test/fetch";
import { navigate } from "../lib/router.svelte";

const summary = {
  shooter: "Ola Nordmann",
  lane: 3,
  scorers: [
    { stage: 1, scorer: "Kari Berg" },
    { stage: 2, scorer: "Range admin" },
  ],
};

let originalFetch: typeof fetch;
let summaryReply: { status?: number; body?: unknown };
beforeEach(() => {
  originalFetch = globalThis.fetch;
  summaryReply = { body: summary };
  stubFetch((call: Call) => {
    if (call.path.endsWith("/summary")) return summaryReply;
    return { body: { name: "Club", has_logo: false, version: 5 } };
  });
});
afterEach(() => {
  globalThis.fetch = originalFetch;
  navigate("/");
});

describe("Scorecard", () => {
  it("shows shooter and scorer names as text next to the card image", async () => {
    navigate("/scorecard/7/3");
    render(Scorecard);
    expect(await screen.findByText("Ola Nordmann")).toBeTruthy();
    expect(screen.getByText(/Kari Berg/)).toBeTruthy();
    expect(screen.getByText(/Range admin/)).toBeTruthy();
    const img = screen.getByRole("img", { name: /Ola Nordmann/ }) as HTMLImageElement;
    expect(img.getAttribute("alt")).toContain("lane 3");
    expect(img.getAttribute("src")).toMatch(/^\/api\/scorer\/scorecard\/7\/3\?format=png&v=/);
    const link = screen.getByRole("link", { name: "Download PNG" });
    expect(link.getAttribute("href")).toBe("/api/scorer/scorecard/7/3?format=png&download=1");
  });

  it("explains when the card is not available", async () => {
    summaryReply = { status: 404, body: { error: "not found" } };
    navigate("/scorecard/7/3");
    render(Scorecard);
    expect(await screen.findByText(/not available/)).toBeTruthy();
    expect(screen.queryByRole("img")).toBeNull();
    expect(screen.queryByRole("link", { name: "Download PNG" })).toBeNull();
  });
});
