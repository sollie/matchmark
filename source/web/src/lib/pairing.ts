import qrcode from "qrcode-generator";

export function pairingUrl(token: string): string {
  return `${window.location.origin}/score#token=${encodeURIComponent(token)}`;
}

export function qrSvg(text: string): string {
  const qr = qrcode(0, "M");
  qr.addData(text);
  qr.make();
  return qr.createSvgTag({ cellSize: 6, margin: 4, scalable: true });
}

const storageKey = "mm_device_token";

export function loadDeviceToken(): string | null {
  const match = /^#token=(.+)$/.exec(window.location.hash);
  if (match) {
    const token = decodeURIComponent(match[1]);
    localStorage.setItem(storageKey, token);
    history.replaceState(null, "", window.location.pathname + window.location.search);
    return token;
  }
  return localStorage.getItem(storageKey);
}

export function forgetDeviceToken(): void {
  localStorage.removeItem(storageKey);
}
