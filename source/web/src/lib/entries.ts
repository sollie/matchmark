import { api } from "./api/client";
import { settle } from "./api/util";
import type { components } from "./api/schema";

type RingCounts = components["schemas"]["RingCounts"];

export type EntryCommands = ReturnType<typeof entryCommands>;

export function entryCommands(matchId: number, reload: () => Promise<void>) {
  const entryPath = (entryID: number) => ({ matchID: matchId, entryID });
  const segmentPath = (entryID: number, segment: number) => ({ ...entryPath(entryID), segment });

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
    unlock: (entryID: number, segment: number) =>
      run(
        api.POST("/api/matches/{matchID}/entries/{entryID}/segments/{segment}/unlock", {
          params: { path: segmentPath(entryID, segment) },
        }),
      ),
    accept: (entryID: number, segment: number) =>
      run(
        api.POST("/api/matches/{matchID}/entries/{entryID}/segments/{segment}/accept", {
          params: { path: segmentPath(entryID, segment) },
        }),
      ),
    lock: (entryID: number, segment: number) =>
      run(
        api.POST("/api/matches/{matchID}/entries/{entryID}/segments/{segment}/lock", {
          params: { path: segmentPath(entryID, segment) },
        }),
      ),
    saveCounts: (entryID: number, segment: number, body: RingCounts) =>
      run(
        api.PUT("/api/matches/{matchID}/entries/{entryID}/segments/{segment}", {
          params: { path: segmentPath(entryID, segment) },
          body,
        }),
      ),
  };
}
