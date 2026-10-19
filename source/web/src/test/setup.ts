const NativeRequest = globalThis.Request;

class AbsoluteRequest extends NativeRequest {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    super(typeof input === "string" ? new URL(input, "http://localhost").toString() : input, init);
  }
}

globalThis.Request = AbsoluteRequest;

import { installFetch } from "./fetch";

installFetch();
