import { api } from "./api/client";

export const admin = $state<{ checked: boolean; authed: boolean; mustChangePassword: boolean }>({
  checked: false,
  authed: false,
  mustChangePassword: false,
});

export async function refreshAdminSession(): Promise<void> {
  const { response } = await api.GET("/api/admin/session");
  admin.mustChangePassword = response.status === 403;
  admin.authed = response.ok || admin.mustChangePassword;
  admin.checked = true;
}

export function markLoggedOut(): void {
  admin.authed = false;
  admin.mustChangePassword = false;
}

export async function logout(): Promise<void> {
  await api.POST("/api/admin/logout");
  admin.authed = false;
  admin.mustChangePassword = false;
}
