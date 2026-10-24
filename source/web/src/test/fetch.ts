export type Call = { method: string; path: string; search: string; body: unknown; authorization: string | null };
type Reply = { status?: number; body?: unknown; headers?: Record<string, string> } | undefined;

let handler: (request: Request) => Promise<Response> = async () => new Response(null, { status: 500 });

export function installFetch() {
  globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) =>
    handler(input instanceof Request ? input : new Request(input, init))) as typeof fetch;
}

export function stubFetch(respond: (call: Call) => Reply = () => undefined) {
  const calls: Call[] = [];
  handler = async (request) => {
    const text = await request.text();
    const call: Call = {
      method: request.method,
      path: new URL(request.url).pathname,
      search: new URL(request.url).search,
      body: text ? JSON.parse(text) : undefined,
      authorization: request.headers.get("Authorization"),
    };
    calls.push(call);
    const reply = respond(call) ?? {};
    return new Response(reply.body === undefined ? null : JSON.stringify(reply.body), {
      status: reply.status ?? 200,
      headers: { "Content-Type": "application/json", ...reply.headers },
    });
  };
  return calls;
}
