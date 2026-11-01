export type Theme = "auto" | "light" | "dark";

export const themeStorageKey = "matchmark-theme";

function read(): Theme {
  try {
    const value = localStorage.getItem(themeStorageKey);
    return value === "light" || value === "dark" ? value : "auto";
  } catch {
    return "auto";
  }
}

export const theme = $state<{ value: Theme }>({ value: read() });

export function setTheme(next: Theme) {
  theme.value = next;
  if (next === "auto") document.documentElement.removeAttribute("data-theme");
  else document.documentElement.setAttribute("data-theme", next);
  try {
    if (next === "auto") localStorage.removeItem(themeStorageKey);
    else localStorage.setItem(themeStorageKey, next);
  } catch {
    return;
  }
}
