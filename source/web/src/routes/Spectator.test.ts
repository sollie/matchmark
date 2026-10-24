import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/svelte";
import Spectator from "./Spectator.svelte";
import { stubFetch } from "../test/fetch";

const standing = (lane: number, states: string[], withdrawn = false) => ({
  lane,
  category: "Open",
  withdrawn,
  total: 0,
  stages: states.map((state, i) => ({ stage: i + 1, accepted: state === "accepted", state })),
});

let originalFetch: typeof fetch;
let send: ((data: unknown) => void) | null;

function install(status: string) {
  const calls = stubFetch(() => ({
    body: { competitions: [{ id: 7, name: "Club Match", date: "2026-05-01", course: "150", status }] },
  }));
  const previous = globalThis.fetch;
  globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(input, init);
    if (new URL(request.url).pathname !== "/api/spectator/competitions/7/stream") return previous(input, init);
    const enc = new TextEncoder();
    let controller!: ReadableStreamDefaultController<Uint8Array>;
    const body = new ReadableStream<Uint8Array>({ start: (c) => (controller = c) });
    send = (data) => controller.enqueue(enc.encode(`event: snapshot\ndata: ${JSON.stringify(data)}\n\n`));
    return Promise.resolve(new Response(body, { headers: { "Content-Type": "text/event-stream" } }));
  }) as typeof fetch;
  return calls;
}

beforeEach(() => {
  originalFetch = globalThis.fetch;
  send = null;
});
afterEach(() => {
  globalThis.fetch = originalFetch;
});

describe("Spectator status row", () => {
  it("shows the status, stage position and accepted count without progress bars", async () => {
    install("active");
    const { container } = render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({
      current_stage: 2,
      total_stages: 10,
      standings: [
        standing(1, ["accepted", "accepted"]),
        standing(2, ["accepted", "awaiting_acceptance"]),
        standing(3, ["accepted", "in_progress"]),
        standing(4, ["accepted", "in_progress"], true),
      ],
    });
    const row = await screen.findByRole("status");
    await waitFor(() => expect(within(row).getByText("Stage 2 of 10")).toBeTruthy());
    expect(within(row).getByText("1/3 accepted")).toBeTruthy();
    expect(within(row).getByText("active")).toBeTruthy();
    expect(within(row).getByText("Live")).toBeTruthy();
    expect(screen.queryByText("Stage progress")).toBeNull();
    expect(container.querySelector("progress")).toBeNull();
  });

  it("shows Finished instead of the stage position", async () => {
    install("finished");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_stage: 10, total_stages: 10, standings: [standing(1, ["accepted"])] });
    const row = await screen.findByRole("status");
    await waitFor(() => expect(within(row).getByText("Finished")).toBeTruthy());
    expect(within(row).queryByText(/Stage \d+ of/)).toBeNull();
    expect(within(row).queryByText(/accepted/)).toBeNull();
  });

  it("links standings rows to the scorecard only once finished", async () => {
    install("finished");
    const { unmount } = render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_stage: 10, total_stages: 10, standings: [standing(1, ["accepted"])] });
    const link = await screen.findByRole("link", { name: "Scorecard for lane 1" });
    expect(link.getAttribute("href")).toMatch(/^\/scorecard\/\d+\/1$/);
    unmount();
    send = null;
    install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    send!({ current_stage: 2, total_stages: 10, standings: [standing(1, ["accepted"])] });
    await screen.findByRole("table");
    expect(screen.queryByRole("link", { name: /Scorecard for lane/ })).toBeNull();
  });

  it("notices a competition finishing from the snapshot without reloading the list", async () => {
    const calls = install("active");
    render(Spectator);
    await screen.findByRole("combobox");
    await waitFor(() => expect(send).not.toBeNull());
    const loads = () => calls.filter((c) => c.path === "/api/spectator/competitions").length;
    const before = loads();
    send!({ current_stage: 2, total_stages: 10, status: "active", standings: [standing(1, ["accepted"])] });
    await screen.findByRole("table");
    expect(screen.queryByRole("link", { name: /Scorecard for lane/ })).toBeNull();
    send!({ current_stage: 10, total_stages: 10, status: "finished", standings: [standing(1, ["accepted"])] });
    expect(await screen.findByRole("link", { name: "Scorecard for lane 1" })).toBeTruthy();
    expect(loads()).toBe(before);
  });
});
