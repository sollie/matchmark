import { api } from "./api/client";

export const admin = $state<{ checked: boolean; authed: boolean }>({ checked: false, authed: false });

export async function refreshAdminSession(): Promise<void> {
  const { response } = await api.GET("/api/admin/session");
  admin.authed = response.ok;
  admin.checked = true;
}

export function markLoggedOut(): void {
  admin.authed = false;
}

export async function logout(): Promise<void> {
  await api.POST("/api/admin/logout");
  admin.authed = false;
}
