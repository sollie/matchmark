import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { autoScroll } from "./autoScroll";

function makeNode(scrollHeight: number, clientHeight: number) {
  const node = document.createElement("div");
  let top = 0;
  const size = { scrollHeight, clientHeight };
  Object.defineProperty(node, "scrollHeight", { get: () => size.scrollHeight });
  Object.defineProperty(node, "clientHeight", { get: () => size.clientHeight });
  Object.defineProperty(node, "scrollTop", {
    get: () => top,
    set: (v: number) => {
      top = v;
    },
  });
  return { node, size };
}

function reducedMotion(matches: boolean) {
  vi.stubGlobal("matchMedia", (query: string) => ({
    matches: matches && query.includes("prefers-reduced-motion"),
  }));
}

beforeEach(() => {
  vi.useFakeTimers();
  reducedMotion(false);
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("autoScroll", () => {
  it("does not scroll when the content fits", () => {
    const { node } = makeNode(300, 300);
    autoScroll(node);
    vi.advanceTimersByTime(60_000);
    expect(node.scrollTop).toBe(0);
  });

  it("scrolls down, pauses, scrolls up, pauses and repeats", () => {
    const { node } = makeNode(600, 500);
    autoScroll(node);
    vi.advanceTimersByTime(3_900);
    expect(node.scrollTop).toBe(0);
    vi.advanceTimersByTime(2_000);
    expect(node.scrollTop).toBeGreaterThan(40);
    expect(node.scrollTop).toBeLessThan(60);
    vi.advanceTimersByTime(2_200);
    expect(node.scrollTop).toBe(100);
    vi.advanceTimersByTime(3_500);
    expect(node.scrollTop).toBe(100);
    vi.advanceTimersByTime(1_500);
    expect(node.scrollTop).toBeLessThan(100);
    vi.advanceTimersByTime(4_000);
    expect(node.scrollTop).toBe(0);
    vi.advanceTimersByTime(2_500);
    expect(node.scrollTop).toBe(0);
    vi.advanceTimersByTime(1_500);
    expect(node.scrollTop).toBeGreaterThan(0);
  });

  it.each(["wheel", "touchstart", "keydown"])("pauses on %s and resumes after idle", (type) => {
    const { node } = makeNode(2000, 500);
    autoScroll(node);
    vi.advanceTimersByTime(6_000);
    node.dispatchEvent(new Event(type));
    node.scrollTop = 300;
    vi.advanceTimersByTime(9_000);
    expect(node.scrollTop).toBe(300);
    vi.advanceTimersByTime(2_000);
    expect(node.scrollTop).toBeGreaterThan(300);
  });

  it("does not scroll under reduced motion", () => {
    reducedMotion(true);
    const { node } = makeNode(2000, 500);
    autoScroll(node);
    vi.advanceTimersByTime(60_000);
    expect(node.scrollTop).toBe(0);
  });

  it("keeps its position when content grows and stops when it fits again", () => {
    const { node, size } = makeNode(1000, 500);
    autoScroll(node);
    vi.advanceTimersByTime(10_000);
    const before = node.scrollTop;
    size.scrollHeight = 1200;
    vi.advanceTimersByTime(1_000);
    expect(node.scrollTop).toBeGreaterThan(before);
    size.scrollHeight = 400;
    vi.advanceTimersByTime(1_000);
    expect(node.scrollTop).toBe(0);
  });

  it("stops after destroy", () => {
    const { node } = makeNode(2000, 500);
    const action = autoScroll(node);
    vi.advanceTimersByTime(6_000);
    action.destroy();
    const stopped = node.scrollTop;
    vi.advanceTimersByTime(10_000);
    expect(node.scrollTop).toBe(stopped);
  });
});
