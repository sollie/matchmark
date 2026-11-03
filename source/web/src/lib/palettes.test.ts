import { describe, expect, it } from "vitest";
import { palettes } from "./palettes";
import css from "../palettes.css?raw";
import goSource from "../../../internal/service/palettes.go?raw";

type Mode = "light" | "dark";
type Tokens = Record<string, Record<Mode, string>>;

function parseBlocks(source: string): Map<string, Tokens> {
  const blocks = new Map<string, Tokens>();
  for (const [, selector, body] of source.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const id = selector.match(/\[data-palette="([a-z-]+)"\]/)?.[1];
    if (!id) continue;
    const tokens: Tokens = {};
    for (const [, name, value] of body.matchAll(/--([a-z-]+):\s*([^;]+);/g)) {
      const pair = value.match(/^light-dark\(\s*(#[0-9a-f]{6})\s*,\s*(#[0-9a-f]{6})\s*\)$/);
      tokens[name] = pair
        ? { light: pair[1], dark: pair[2] }
        : { light: value.trim(), dark: value.trim() };
    }
    blocks.set(id, tokens);
  }
  return blocks;
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

describe("palette lists", () => {
  const ids = palettes.map((p) => p.id);
  const blocks = parseBlocks(css);

  it("the CSS blocks match the TypeScript list", () => {
    expect([...blocks.keys()].sort()).toEqual([...ids].sort());
  });

  it("the Go allow-list matches the TypeScript list", () => {
    const list = goSource.match(/var Palettes = \[\]string\{([^}]*)\}/)![1];
    const goIds = [...list.matchAll(/"([a-z-]+)"/g)].map((m) => m[1]);
    expect(goIds.sort()).toEqual([...ids].sort());
  });

  it("every palette defines the same tokens", () => {
    const reference = Object.keys(blocks.get("matchmark")!).sort();
    for (const [id, tokens] of blocks) {
      expect(Object.keys(tokens).sort(), id).toEqual(reference);
    }
  });
});

const pairs: { fg: string; bg: string; min: number }[] = [
  { fg: "text", bg: "surface", min: 4.5 },
  { fg: "text", bg: "bg", min: 4.5 },
  { fg: "text-muted", bg: "surface", min: 4.5 },
  { fg: "text-muted", bg: "bg", min: 4.4 },
  { fg: "link", bg: "surface", min: 4.5 },
  { fg: "on-brand", bg: "brand", min: 3.8 },
  { fg: "on-danger", bg: "danger", min: 4.5 },
  { fg: "success-fg", bg: "success-bg", min: 4.5 },
  { fg: "warning-fg", bg: "warning-bg", min: 4.5 },
  { fg: "danger-fg", bg: "danger-bg", min: 4.5 },
  { fg: "neutral-fg", bg: "neutral-bg", min: 4.5 },
  { fg: "focus", bg: "surface", min: 3 },
];

describe("palette contrast", () => {
  const blocks = parseBlocks(css);
  const rows = [...blocks].flatMap(([id, tokens]) =>
    (["light", "dark"] as Mode[]).flatMap((mode) =>
      pairs.map((p) => ({
        id,
        mode,
        ...p,
        ratio: contrast(tokens[p.fg][mode], tokens[p.bg][mode]),
      })),
    ),
  );

  it("reports the ratios", () => {
    console.table(rows.map((r) => ({ ...r, ratio: r.ratio.toFixed(2) })));
  });

  it.each(rows)("$id $mode $fg on $bg is at least $min", (row) => {
    expect(row.ratio).toBeGreaterThanOrEqual(row.min);
  });
});
