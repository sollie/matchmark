import { expect, test } from "./helpers";

test.describe("overlay accessibility", () => {
  test.beforeEach(async ({ page, adminPage }) => {
    void adminPage;
    await page.setViewportSize({ width: 390, height: 800 });
    await page.goto("/admin");
  });

  test("the More drawer traps focus and returns it to the tab on Escape @smoke", async ({
    page,
  }) => {
    const more = page.getByRole("navigation", { name: "Admin tabs" }).getByRole("button", {
      name: "More",
    });
    await more.click();
    const drawer = page.getByRole("dialog", { name: "More" });
    await expect(drawer).toBeVisible();
    await expect(drawer.getByRole("button", { name: "Dismiss" })).toBeVisible();

    for (let i = 0; i < 12; i++) {
      await page.keyboard.press("Tab");
      const inside = await drawer.evaluate((el) => el.contains(document.activeElement));
      expect(inside).toBe(true);
    }

    await page.keyboard.press("Escape");
    await expect(drawer).toHaveCount(0);
    await expect(more).toBeFocused();
  });

  test("overlay animations are disabled under reduced motion", async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page
      .getByRole("navigation", { name: "Admin tabs" })
      .getByRole("button", { name: "More" })
      .click();
    const drawer = page.getByRole("dialog", { name: "More" });
    await expect(drawer).toBeVisible();
    const names = await drawer.evaluate((el) => [
      getComputedStyle(el).animationName,
      getComputedStyle(el.previousElementSibling!).animationName,
    ]);
    expect(names).toEqual(["none", "none"]);
  });
});
