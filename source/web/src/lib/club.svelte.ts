import { api } from "./api/client";
import { applyPalette } from "./palettes";

export const club = $state({
  name: "",
  palette: "matchmark",
  hasLogo: false,
  version: 0,
  unclassifiedClass: "Master",
});

export async function refreshClub() {
  const { data } = await api.GET("/api/club");
  if (!data) return;
  club.name = data.name;
  club.palette = data.palette;
  applyPalette(data.palette);
  club.hasLogo = data.has_logo;
  club.version = data.version;
  club.unclassifiedClass = data.unclassified_class;
}

export function clubLogoUrl(version: number): string {
  return `/api/club/logo?v=${version}`;
}
