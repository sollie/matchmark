import type { components } from "./api/schema";

export type Classification = components["schemas"]["Classification"];
export type Classifications = components["schemas"]["Shooter"]["classifications"];

const categoryOrder = [
  "Revolver 1500",
  "Pistol 1500",
  "Revolver Optical Sight 1500",
  "Pistol Optical Sight 1500",
  "Carbine 1500",
];

export function classChips(classifications: Classifications | undefined) {
  return categoryOrder.flatMap((category) => {
    const c = classifications?.[category as keyof Classifications];
    return c && c.value > 0 ? [{ category, name: c.name }] : [];
  });
}
