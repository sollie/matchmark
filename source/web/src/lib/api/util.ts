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
