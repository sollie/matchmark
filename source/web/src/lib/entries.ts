import { api } from "./api/client";
import { settle } from "./api/util";
import type { components } from "./api/schema";

type RingCounts = components["schemas"]["RingCounts"];

export type EntryCommands = ReturnType<typeof entryCommands>;

export function entryCommands(matchId: number, reload: () => Promise<void>) {
  const entryPath = (entryID: number) => ({ matchID: matchId, entryID });
  const stagePath = (entryID: number, stage: number) => ({ ...entryPath(entryID), stage });

  async function run(
    call: Promise<{ error?: unknown; response: Response }>,
  ): Promise<string | null> {
    const result = await settle(call);
    if (!result.ok) return result.error;
    await reload();
    return null;
  }

  return {
    add: (body: { shooter_id: number; category: string; lane: number }) =>
      run(
        api.POST("/api/matches/{matchID}/entries", {
          params: { path: { matchID: matchId } },
          body,
        }),
      ),
    update: (entryID: number, body: { category: string; lane: number }) =>
      run(
        api.PUT("/api/matches/{matchID}/entries/{entryID}", {
          params: { path: entryPath(entryID) },
          body,
        }),
      ),
    withdraw: (entryID: number) =>
      run(
        api.POST("/api/matches/{matchID}/entries/{entryID}/withdraw", {
          params: { path: entryPath(entryID) },
        }),
      ),
    remove: (entryID: number) =>
      run(
        api.DELETE("/api/matches/{matchID}/entries/{entryID}", {
          params: { path: entryPath(entryID) },
        }),
      ),
    unlock: (entryID: number, stage: number) =>
      run(
        api.POST("/api/matches/{matchID}/entries/{entryID}/stages/{stage}/unlock", {
          params: { path: stagePath(entryID, stage) },
        }),
      ),
    accept: (entryID: number, stage: number) =>
      run(
        api.POST("/api/matches/{matchID}/entries/{entryID}/stages/{stage}/accept", {
          params: { path: stagePath(entryID, stage) },
        }),
      ),
    lock: (entryID: number, stage: number) =>
      run(
        api.POST("/api/matches/{matchID}/entries/{entryID}/stages/{stage}/lock", {
          params: { path: stagePath(entryID, stage) },
        }),
      ),
    saveCounts: (entryID: number, stage: number, body: RingCounts) =>
      run(
        api.PUT("/api/matches/{matchID}/entries/{entryID}/stages/{stage}", {
          params: { path: stagePath(entryID, stage) },
          body,
        }),
      ),
  };
}
