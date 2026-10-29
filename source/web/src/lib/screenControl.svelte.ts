import { api } from "./api/client";
import { openStream, type StreamHandle, type StreamState } from "./sse";
import type { components } from "./api/schema";

type ScreenState = components["schemas"]["ScreenState"];

const storageKey = "matchmark.screen";

type Stored = { token: string; code: string };

function load(): Stored | null {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    return typeof parsed?.token === "string" && typeof parsed?.code === "string" ? parsed : null;
  } catch {
    return null;
  }
}

export class ScreenControl {
  registration = $state<Stored | null>(load());
  assignment = $state<ScreenState | null>(null);
  connection = $state<StreamState>("connecting");
  notice = $state("");
  error = $state("");
  busy = $state(false);

  managed = $derived(this.registration !== null);
  private stream: StreamHandle | null = null;

  start() {
    this.stop();
    const registration = this.registration;
    if (!registration) return;
    let own: StreamHandle | null = null;
    own = openStream({
      url: "/api/spectator/screens/stream",
      headers: { Authorization: `Bearer ${registration.token}` },
      onEvent: (event, data) => {
        if (event === "snapshot") this.assignment = JSON.parse(data);
      },
      onState: (state) => {
        this.connection = state;
        if (state === "ended" && this.stream === own) {
          this.revert(
            "This screen was removed by an administrator and is no longer remote controlled.",
          );
        }
      },
    });
    this.stream = own;
  }

  stop() {
    const stream = this.stream;
    this.stream = null;
    stream?.close();
  }

  private revert(notice: string) {
    this.stop();
    localStorage.removeItem(storageKey);
    this.registration = null;
    this.assignment = null;
    this.notice = notice;
  }

  async enable() {
    if (this.busy) return;
    this.busy = true;
    this.error = "";
    this.notice = "";
    const { data, error, response } = await api.POST("/api/spectator/screens");
    this.busy = false;
    if (!data) {
      this.error =
        response?.status === 429
          ? "Too many attempts. Try again later."
          : (error?.error ?? "Could not enable remote control.");
      return;
    }
    const stored = { token: data.token, code: data.code };
    localStorage.setItem(storageKey, JSON.stringify(stored));
    this.registration = stored;
    this.start();
  }

  async leave() {
    const registration = this.registration;
    if (!registration || this.busy) return;
    this.busy = true;
    await api.DELETE("/api/spectator/screens/self", {
      headers: { Authorization: `Bearer ${registration.token}` },
    });
    this.busy = false;
    this.revert("");
  }
}
