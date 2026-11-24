import { beforeEach, describe, expect, it, vi } from "vitest";
import { Command } from "./command.svelte";
import { admin } from "./session.svelte";

const reply = (status: number, body: { data?: unknown; error?: unknown } = {}) =>
  Promise.resolve({ ...body, response: new Response(null, { status }) });

describe("Command", () => {
  const reload = vi.fn(async () => {});

  beforeEach(() => {
    reload.mockClear();
  });

  it("sets the notice and reloads after a successful call", async () => {
    const command = new Command(reload);
    const result = await command.run(reply(200, { data: { id: 1 } }), "Saved.");
    expect(result).toEqual({ ok: true, data: { id: 1 } });
    expect(command.notice).toBe("Saved.");
    expect(command.error).toBe("");
    expect(reload).toHaveBeenCalledOnce();
  });

  it("builds the notice from the response data", async () => {
    const command = new Command(reload);
    await command.run(
      reply(200, { data: { username: "ann" } }),
      (d: { username: string }) => `User ${d.username} created.`,
    );
    expect(command.notice).toBe("User ann created.");
  });

  it("sets the error and skips the reload after a failed call", async () => {
    const command = new Command(reload);
    const result = await command.run(reply(409, { error: { error: "Name taken" } }), "Saved.");
    expect(result.ok).toBe(false);
    expect(command.error).toBe("Name taken");
    expect(command.notice).toBe("");
    expect(reload).not.toHaveBeenCalled();
  });

  it("clears earlier messages before running", async () => {
    const command = new Command(reload);
    await command.run(reply(200), "First.");
    await command.run(reply(500));
    expect(command.notice).toBe("");
    expect(command.error).toBe("Request failed (500)");
    await command.run(reply(200));
    expect(command.error).toBe("");
  });

  it("falls back to a generic message and marks the session logged out on 401", async () => {
    const command = new Command(reload);
    admin.authed = true;
    await command.run(reply(401));
    expect(command.error).toBe("Request failed (401)");
    expect(admin.authed).toBe(false);
  });

  it("clear resets both messages", async () => {
    const command = new Command(reload);
    await command.run(reply(200), "Done.");
    command.fail("Broken");
    command.clear();
    expect(command.error).toBe("");
    expect(command.notice).toBe("");
  });
});
