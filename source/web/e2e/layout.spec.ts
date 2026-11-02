import { expect, openAccountLink, test } from "./helpers";

test.describe("layout", () => {
  test("the users table stacks without horizontal overflow at 375px", async ({
    adminPage: page,
  }) => {
    await page.setViewportSize({ width: 375, height: 800 });
    await openAccountLink(page, "Users");
    await expect(page.getByRole("heading", { name: "Users" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Reset password" }).first()).toBeVisible();
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
  });

  test("tables keep their column layout on a wide monitor", async ({ adminPage: page }) => {
    await page.setViewportSize({ width: 1920, height: 1000 });
    await openAccountLink(page, "Users");
    await expect(page.getByRole("columnheader", { name: "Username" })).toBeVisible();
  });
});
