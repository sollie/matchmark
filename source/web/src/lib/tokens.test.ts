import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";

const css = readFileSync("src/tokens.css", "utf8");

type Mode = "light" | "dark";

function parseTokens(source: string): Record<string, Record<Mode, string>> {
  const root = source.match(/:root\s*\{([^}]*)\}/)![1];
  const tokens: Record<string, Record<Mode, string>> = {};
  for (const [, name, value] of root.matchAll(/--([a-z-]+):\s*([^;]+);/g)) {
    const pair = value.match(/^light-dark\(\s*(#[0-9a-f]{6})\s*,\s*(#[0-9a-f]{6})\s*\)$/);
    if (pair) tokens[name] = { light: pair[1], dark: pair[2] };
  }
  return tokens;
}

function luminance(hex: string): number {
  const [r, g, b] = [1, 3, 5]
    .map((i) => parseInt(hex.slice(i, i + 2), 16) / 255)
    .map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

const pairs: { fg: string; bg: string; min: number }[] = [
  { fg: "text", bg: "surface", min: 4.5 },
  { fg: "text", bg: "bg", min: 4.5 },
  { fg: "text-muted", bg: "surface", min: 4.5 },
  { fg: "text-muted", bg: "bg", min: 4.4 },
  { fg: "text-muted", bg: "surface-alt", min: 4.5 },
  { fg: "link", bg: "surface", min: 4.5 },
  { fg: "link", bg: "brand-tint", min: 4.5 },
  { fg: "brand", bg: "surface", min: 4.5 },
  { fg: "on-brand", bg: "brand", min: 3.8 },
  { fg: "on-danger", bg: "danger", min: 4.5 },
  { fg: "success-fg", bg: "success-bg", min: 4.5 },
  { fg: "warning-fg", bg: "warning-bg", min: 4.5 },
  { fg: "danger-fg", bg: "danger-bg", min: 4.5 },
  { fg: "neutral-fg", bg: "neutral-bg", min: 4.5 },
  { fg: "focus", bg: "surface", min: 3 },
];

describe("brand token contrast", () => {
  const tokens = parseTokens(css);
  const rows = (["light", "dark"] as Mode[]).flatMap((mode) =>
    pairs.map((p) => ({
      mode,
      ...p,
      ratio: contrast(tokens[p.fg][mode], tokens[p.bg][mode]),
    })),
  );

  it.each(rows)("$mode $fg on $bg is at least $min", (row) => {
    expect(row.ratio).toBeGreaterThanOrEqual(row.min);
  });
});
