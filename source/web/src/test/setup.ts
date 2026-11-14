const NativeRequest = globalThis.Request;

class AbsoluteRequest extends NativeRequest {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    super(typeof input === "string" ? new URL(input, "http://localhost").toString() : input, init);
  }
}

globalThis.Request = AbsoluteRequest;

import { installFetch } from "./fetch";

installFetch();

HTMLDialogElement.prototype.showModal = function showModal() {
  this.setAttribute("open", "");
};
HTMLDialogElement.prototype.close = function close() {
  if (!this.hasAttribute("open")) return;
  this.removeAttribute("open");
  this.dispatchEvent(new Event("close"));
};

const floatingStyle = document.createElement("style");
floatingStyle.textContent =
  "[data-bits-floating-content-wrapper] { visibility: visible !important; }";
document.head.appendChild(floatingStyle);

Element.prototype.hasPointerCapture ??= () => false;
Element.prototype.setPointerCapture ??= () => {};
Element.prototype.releasePointerCapture ??= () => {};
Element.prototype.scrollIntoView ??= () => {};
