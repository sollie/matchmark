export type StreamState = "connecting" | "live" | "stale" | "paused" | "ended";

export type StreamOptions = {
  url: string;
  headers?: Record<string, string>;
  onEvent: (event: string, data: string) => void;
  onState?: (state: StreamState) => void;
  terminalEvents?: string[];
  backoff?: { initialMs: number; maxMs: number };
  staleAfterMs?: number;
  fetch?: typeof fetch;
  random?: () => number;
};

export type StreamHandle = { close: () => void };

const terminalStatuses = new Set([401, 403, 404, 409, 410]);
const stableAfterMs = 5000;

export function createSseParser(emit: (event: string, data: string) => void) {
  let buffer = "";
  let event = "";
  let data: string[] = [];

  function line(text: string) {
    if (text === "") {
      if (data.length > 0) emit(event || "message", data.join("\n"));
      event = "";
      data = [];
      return;
    }
    if (text.startsWith(":")) return;
    const colon = text.indexOf(":");
    const field = colon === -1 ? text : text.slice(0, colon);
    let value = colon === -1 ? "" : text.slice(colon + 1);
    if (value.startsWith(" ")) value = value.slice(1);
    if (field === "event") event = value;
    else if (field === "data") data.push(value);
  }

  return {
    push(chunk: string) {
      buffer += chunk;
      let match: RegExpExecArray | null;
      while ((match = /\r\n|\n|\r/.exec(buffer))) {
        if (match[0] === "\r" && match.index === buffer.length - 1) return;
        const text = buffer.slice(0, match.index);
        buffer = buffer.slice(match.index + match[0].length);
        line(text);
      }
    },
  };
}

export function openStream(opts: StreamOptions): StreamHandle {
  const backoff = opts.backoff ?? { initialMs: 1000, maxMs: 30000 };
  const staleAfterMs = opts.staleAfterMs ?? 60000;
  const random = opts.random ?? Math.random;
  const terminal = new Set(["revoked", ...(opts.terminalEvents ?? [])]);

  let state: StreamState | null = null;
  let closed = false;
  let failures = 0;
  let controller: AbortController | null = null;
  let retryTimer: ReturnType<typeof setTimeout> | undefined;
  let staleTimer: ReturnType<typeof setTimeout> | undefined;

  function setState(next: StreamState) {
    if (state === next) return;
    state = next;
    opts.onState?.(next);
  }

  function armStale() {
    clearTimeout(staleTimer);
    staleTimer = setTimeout(() => controller?.abort(), staleAfterMs);
  }

  function stop(next: StreamState) {
    clearTimeout(retryTimer);
    clearTimeout(staleTimer);
    controller?.abort();
    controller = null;
    setState(next);
  }

  function scheduleRetry(retryAfterMs = 0) {
    const ceiling = Math.min(backoff.maxMs, backoff.initialMs * 2 ** failures);
    failures++;
    clearTimeout(retryTimer);
    const delay = Math.max(retryAfterMs, Math.round(ceiling * (0.5 + 0.5 * random())));
    retryTimer = setTimeout(connect, delay);
  }

  function lost(retryAfterMs = 0) {
    if (closed || document.hidden) return;
    setState("stale");
    scheduleRetry(retryAfterMs);
  }

  async function connect() {
    if (closed || document.hidden) return;
    const own = new AbortController();
    controller = own;
    if (state !== "stale") setState("connecting");
    armStale();
    let ended = false;
    let retryAfterMs = 0;
    let liveSince = 0;
    try {
      const response = await (opts.fetch ?? globalThis.fetch)(opts.url, {
        headers: { Accept: "text/event-stream", ...opts.headers },
        signal: own.signal,
        credentials: "same-origin",
      });
      if (terminalStatuses.has(response.status)) {
        ended = true;
        stop("ended");
        return;
      }
      if (response.status === 503) {
        const seconds = Number(response.headers.get("Retry-After"));
        if (Number.isFinite(seconds) && seconds > 0) retryAfterMs = seconds * 1000;
      }
      if (!response.ok || !response.body) throw new Error(`stream status ${response.status}`);
      liveSince = Date.now();
      setState("live");
      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      const parser = createSseParser((event, data) => {
        if (terminal.has(event)) {
          ended = true;
          opts.onEvent(event, data);
          stop("ended");
          return;
        }
        opts.onEvent(event, data);
      });
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        armStale();
        parser.push(decoder.decode(value, { stream: true }));
        if (ended) return;
      }
    } catch {
      if (own.signal.aborted && controller !== own) return;
    }
    if (ended || controller !== own) return;
    if (liveSince > 0 && Date.now() - liveSince >= stableAfterMs) failures = 0;
    clearTimeout(staleTimer);
    lost(retryAfterMs);
  }

  function onVisibility() {
    if (closed || state === "ended") return;
    if (document.hidden) {
      stop("paused");
    } else if (state === "paused") {
      failures = 0;
      void connect();
    }
  }

  document.addEventListener("visibilitychange", onVisibility);
  if (document.hidden) setState("paused");
  else void connect();

  return {
    close() {
      closed = true;
      document.removeEventListener("visibilitychange", onVisibility);
      stop("ended");
    },
  };
}
