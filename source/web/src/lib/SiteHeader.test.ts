import { afterEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import SiteHeader from "./SiteHeader.svelte";
import { router } from "./router.svelte";
import { admin } from "./session.svelte";

afterEach(() => {
  router.path = "/";
  admin.authed = false;
  admin.isAdmin = false;
  admin.canDownloadBackup = false;
});

describe("SiteHeader", () => {
  it("opens a menu with the theme group and an admin link on public routes", async () => {
    const user = userEvent.setup();
    render(SiteHeader);
    await user.click(screen.getByRole("button", { name: "Menu" }));
    expect(screen.getByRole("radiogroup", { name: "Theme" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Admin" }).getAttribute("href")).toBe("/admin");
  });

  it("offers only the theme group on the scorer route", async () => {
    router.path = "/score";
    const user = userEvent.setup();
    render(SiteHeader);
    await user.click(screen.getByRole("button", { name: "Menu" }));
    expect(screen.getByRole("radiogroup", { name: "Theme" })).toBeTruthy();
    expect(screen.queryByRole("link", { name: "Admin" })).toBeNull();
  });

  describe("on admin routes", () => {
    function signIn(isAdmin: boolean, canDownloadBackup = false) {
      router.path = "/admin";
      Object.assign(admin, { authed: true, username: "pat", isAdmin, canDownloadBackup });
    }

    it("shows the account menu instead of the sign-in link, closed on Escape", async () => {
      signIn(true, true);
      const user = userEvent.setup();
      render(SiteHeader);
      expect(screen.queryByRole("button", { name: "Menu" })).toBeNull();
      const trigger = screen.getByRole("button", { name: "Account menu" });
      expect(trigger.getAttribute("aria-expanded")).toBe("false");
      await user.click(trigger);
      expect(trigger.getAttribute("aria-expanded")).toBe("true");
      for (const name of ["Users", "Club", "Audit log", "Backup", "Change password"])
        expect(screen.getByRole("link", { name })).toBeTruthy();
      await user.keyboard("{Escape}");
      expect(trigger.getAttribute("aria-expanded")).toBe("false");
    });

    it("limits regular users to the account group", async () => {
      signIn(false);
      const user = userEvent.setup();
      render(SiteHeader);
      await user.click(screen.getByRole("button", { name: "Account menu" }));
      expect(screen.getByRole("link", { name: "Change password" })).toBeTruthy();
      expect(screen.queryByRole("link", { name: "Users" })).toBeNull();
      expect(screen.queryByRole("link", { name: "Backup" })).toBeNull();
    });

    it("marks the trigger current on menu-owned pages", () => {
      signIn(true);
      router.path = "/admin/audit";
      render(SiteHeader);
      expect(
        screen.getByRole("button", { name: "Account menu" }).getAttribute("aria-current"),
      ).toBe("page");
    });
  });
});
