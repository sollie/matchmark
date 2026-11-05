import type { components } from "./api/schema";

export type Classification = components["schemas"]["Classification"];
export type Classifications = components["schemas"]["Shooter"]["classifications"];

const categoryOrder = ["Revolver", "Pistol", "Revolver OS", "Pistol OS", "Carbine"];

export function classChips(classifications: Classifications | undefined) {
  return categoryOrder.flatMap((category) => {
    const c = classifications?.[category as keyof Classifications];
    return c && c.value > 0 ? [{ category, name: c.name }] : [];
  });
}
