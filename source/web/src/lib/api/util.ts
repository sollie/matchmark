import { markLoggedOut } from "../session.svelte";

export function errorMessage(error: unknown, fallback = "Request failed"): string {
  if (error && typeof error === "object" && "error" in error && typeof error.error === "string") {
    return error.error;
  }
  return fallback;
}

export function failure(result: { error?: unknown; response: Response }): string {
  if (result.response.status === 401) markLoggedOut();
  return errorMessage(result.error, `Request failed (${result.response.status})`);
}

export type Settled<T> = { ok: true; data: T } | { ok: false; error: string; status: number };

export async function settle<T>(
  call: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<Settled<T>> {
  const result = await call;
  if (result.response.ok) return { ok: true, data: result.data as T };
  return { ok: false, error: failure(result), status: result.response.status };
}
