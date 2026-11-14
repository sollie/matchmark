import { render, screen } from "@testing-library/svelte";
import { beforeEach, describe, expect, it } from "vitest";
import { router } from "../lib/router.svelte";
import { admin } from "../lib/session.svelte";
import AdminNav from "./AdminNav.svelte";

describe("AdminNav", () => {
  beforeEach(() => {
    router.path = "/admin";
  });

  it("lists every area for admins and marks the current one", () => {
    Object.assign(admin, { authed: true, isAdmin: true, canDownloadBackup: true });
    router.path = "/admin/audit";
    render(AdminNav);
    for (const name of [
      "Matches",
      "Shooters",
      "Users",
      "Screens",
      "Club",
      "Audit log",
      "Backup",
      "WAID",
      "Change password",
    ])
      expect(screen.getByRole("link", { name })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Audit log" }).getAttribute("aria-current")).toBe(
      "page",
    );
    expect(screen.getByRole("link", { name: "Matches" }).getAttribute("aria-current")).toBeNull();
  });

  it("limits regular users to matches, shooters and their account", () => {
    Object.assign(admin, { authed: true, isAdmin: false, canDownloadBackup: false });
    render(AdminNav);
    expect(screen.getByRole("link", { name: "Change password" })).toBeTruthy();
    expect(screen.queryByRole("link", { name: "Users" })).toBeNull();
    expect(screen.queryByRole("link", { name: "Backup" })).toBeNull();
  });
});
