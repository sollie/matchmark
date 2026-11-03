export const palettes = [
  { id: "matchmark", name: "MatchMark" },
  { id: "arctic-blue", name: "Arctic Blue" },
  { id: "nordic-forest", name: "Nordic Forest" },
  { id: "royal-competition", name: "Royal Competition" },
  { id: "copper-graphite", name: "Copper & Graphite" },
  { id: "graphite-lime", name: "Graphite & Lime" },
] as const;

export type PaletteId = (typeof palettes)[number]["id"];

export const defaultPalette: PaletteId = "matchmark";

export const paletteStorageKey = "matchmark-palette";

export function isPaletteId(value: unknown): value is PaletteId {
  return palettes.some((p) => p.id === value);
}

export function applyPalette(id: string) {
  const next = isPaletteId(id) ? id : defaultPalette;
  document.documentElement.setAttribute("data-palette", next);
  try {
    localStorage.setItem(paletteStorageKey, next);
  } catch {}
}
