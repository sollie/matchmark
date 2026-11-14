import { expect, openAccountLink, test } from "./helpers";

const viewports = [
  { name: "phone", width: 390, height: 800 },
  { name: "tablet", width: 820, height: 1000 },
  { name: "desktop", width: 1440, height: 900 },
];

test.describe("admin shell", () => {
  for (const vp of viewports) {
    test(`reaches every admin area on ${vp.name}`, async ({ page, adminPage }) => {
      void adminPage;
      await page.setViewportSize({ width: vp.width, height: vp.height });
      await page.goto("/admin");
      const sidebar = page.getByRole("navigation", { name: "Admin sections" });
      const tabs = page.getByRole("navigation", { name: "Admin tabs" });
      if (vp.width >= 820) {
        await expect(sidebar).toBeVisible();
        await expect(tabs).toBeHidden();
        await expect(sidebar.getByRole("link", { name: "Matches" })).toHaveAttribute(
          "aria-current",
          "page",
        );
        await sidebar.getByRole("link", { name: "Shooters" }).click();
        await expect(page.getByRole("heading", { name: "Shooters" })).toBeVisible();
        await expect(sidebar.getByRole("link", { name: "Shooters" })).toHaveAttribute(
          "aria-current",
          "page",
        );
      } else {
        await expect(sidebar).toBeHidden();
        await expect(tabs).toBeVisible();
        await tabs.getByRole("link", { name: "Shooters" }).click();
        await expect(page.getByRole("heading", { name: "Shooters" })).toBeVisible();
        await expect(tabs.getByRole("link", { name: "Shooters" })).toHaveAttribute(
          "aria-current",
          "page",
        );
      }

      for (const [link, heading] of [
        ["Users", "Users"],
        ["Screens", "Screens"],
        ["Club", "Club settings"],
        ["Audit log", "Audit log"],
        ["WAID", "Account"],
        ["Change password", "Change password"],
      ]) {
        await openAccountLink(page, link);
        await expect(page.getByRole("heading", { name: heading, exact: true })).toBeVisible();
      }

      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      );
      expect(overflow).toBeLessThanOrEqual(0);
    });
  }

  test("the sign-in form shows its error state", async ({ browser, baseURL }) => {
    const anon = await (await browser.newContext({ baseURL })).newPage();
    await anon.goto("/admin");
    await anon.getByLabel("Username").fill("nobody");
    await anon.getByLabel("Password", { exact: true }).fill("not-a-real-password");
    await anon.getByRole("button", { name: "Sign in" }).click();
    await expect(anon.getByRole("alert")).toBeVisible();
    await anon.close();
  });
});
