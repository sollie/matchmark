import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/svelte";
import Scorecard from "./Scorecard.svelte";
import { stubFetch, type Call } from "../test/fetch";
import { navigate } from "../lib/router.svelte";

const summary = {
  shooter: "Ola Nordmann",
  lane: 3,
  scorers: [
    { segment: 1, segment_name: "Match 1", scorer: "Kari Berg" },
    { segment: 2, segment_name: "Match 2", scorer: "Range admin" },
  ],
};

let originalFetch: typeof fetch;
let summaryReply: { status?: number; body?: unknown; headers?: Record<string, string> };
let imageReply: { status?: number; body?: unknown; headers?: Record<string, string> };
beforeEach(() => {
  originalFetch = globalThis.fetch;
  summaryReply = { body: summary };
  imageReply = { status: 500 };
  stubFetch((call: Call) => {
    if (call.path.endsWith("/summary")) return summaryReply;
    if (call.path.startsWith("/api/scorer/scorecard/")) return imageReply;
    return { body: { name: "Club", has_logo: false, version: 5 } };
  });
});
afterEach(() => {
  vi.useRealTimers();
  globalThis.fetch = originalFetch;
  navigate("/");
});

describe("Scorecard", () => {
  it("shows shooter and scorer names as text next to the card image", async () => {
    navigate("/scorecard/7/3");
    render(Scorecard);
    expect(await screen.findByText("Ola Nordmann")).toBeTruthy();
    expect(screen.getByText("Match 1 scored by Kari Berg")).toBeTruthy();
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

  it("asks to try again shortly when the render limits answer 429, and retries after Retry-After", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    summaryReply = {
      status: 429,
      body: { error: "too many requests" },
      headers: { "Retry-After": "2" },
    };
    navigate("/scorecard/7/3");
    render(Scorecard);
    expect(await screen.findByText(/try again shortly/i)).toBeTruthy();
    expect(screen.getByText(/2 seconds/)).toBeTruthy();
    expect(screen.queryByText(/not available/)).toBeNull();
    summaryReply = { body: summary };
    await vi.advanceTimersByTimeAsync(2100);
    expect(await screen.findByText("Ola Nordmann")).toBeTruthy();
  });

  it("asks to try again shortly when the server is busy", async () => {
    summaryReply = { status: 503, body: { error: "server busy" }, headers: { "Retry-After": "1" } };
    navigate("/scorecard/7/3");
    render(Scorecard);
    expect(await screen.findByText(/try again shortly/i)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
  });

  it("explains a rate-limited card image instead of a missing card", async () => {
    imageReply = {
      status: 429,
      body: { error: "too many requests" },
      headers: { "Retry-After": "5" },
    };
    navigate("/scorecard/7/3");
    render(Scorecard);
    await fireEvent.error(await screen.findByRole("img", { name: /Ola Nordmann/ }));
    expect(await screen.findByText(/try again shortly/i)).toBeTruthy();
    expect(screen.queryByText(/not available/)).toBeNull();
  });

  it("reports an image that cannot be loaded and offers a retry", async () => {
    navigate("/scorecard/7/3");
    render(Scorecard);
    await fireEvent.error(await screen.findByRole("img", { name: /Ola Nordmann/ }));
    expect(await screen.findByText(/could not be loaded/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
  });
});
