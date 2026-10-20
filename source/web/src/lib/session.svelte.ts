import { api } from "./api/client";

export const admin = $state<{
  checked: boolean;
  authed: boolean;
  mustChangePassword: boolean;
  username: string;
  isAdmin: boolean;
  canDownloadBackup: boolean;
}>({
  checked: false,
  authed: false,
  mustChangePassword: false,
  username: "",
  isAdmin: false,
  canDownloadBackup: false,
});

export async function refreshAdminSession(): Promise<void> {
  const { data, response } = await api.GET("/api/admin/session");
  admin.mustChangePassword = response.status === 403;
  admin.authed = response.ok || admin.mustChangePassword;
  admin.username = data?.username ?? "";
  admin.isAdmin = data?.role === "admin";
  admin.canDownloadBackup = data?.can_download_backup ?? false;
  admin.checked = true;
}

function clear(): void {
  admin.authed = false;
  admin.mustChangePassword = false;
  admin.username = "";
  admin.isAdmin = false;
  admin.canDownloadBackup = false;
}

export function markLoggedOut(): void {
  clear();
}

export async function logout(): Promise<void> {
  await api.POST("/api/admin/logout");
  clear();
}
