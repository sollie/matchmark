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
