const collapse = (s: string) => s.split(/\s+/).filter(Boolean).join(" ");

const simpleUpper = (ch: string) => {
  const u = ch.toUpperCase();
  return [...u].length === 1 ? u : ch;
};

const simpleLower = (ch: string) => {
  const l = ch.toLowerCase();
  return [...l].length === 1 ? l : ch;
};

const startsWord = (ch: string) => ch === " " || ch === "-" || ch === "'" || ch === "’";

export function liveLastName(s: string): string {
  return [...s].map(simpleUpper).join("");
}

export function liveFirstName(s: string): string {
  let upper = true;
  let out = "";
  for (const ch of s) {
    out += upper ? simpleUpper(ch) : simpleLower(ch);
    upper = startsWord(ch) || /\s/.test(ch);
  }
  return out;
}

export const normalizeLastName = (s: string): string => liveLastName(collapse(s));

export const normalizeFirstName = (s: string): string => liveFirstName(collapse(s));
