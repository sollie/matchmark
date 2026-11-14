import type { Page } from "@playwright/test";
import { expect, test } from "./helpers";

const lightBackground = "rgb(241, 243, 246)";
const darkBackground = "rgb(14, 19, 27)";

async function background(page: Page) {
  return page.evaluate(() => getComputedStyle(document.documentElement).backgroundColor);
}

async function chooseTheme(page: Page, name: "Auto" | "Light" | "Dark") {
  const trigger = page.getByRole("button", { name: /^(Menu|Account menu)$/ });
  if ((await trigger.getAttribute("aria-expanded")) !== "true") await trigger.click();
  await page.getByRole("radiogroup", { name: "Mode" }).getByLabel(name).check();
}

test.describe("theme", () => {
  test.use({ colorScheme: "dark" });

  test("Auto follows the OS, Light overrides it and the choice persists", async ({ page }) => {
    await page.goto("/");
    expect(await background(page)).toBe(darkBackground);
    await expect(page.locator("html")).not.toHaveAttribute("data-theme", /.*/);

    await chooseTheme(page, "Light");
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    expect(await background(page)).toBe(lightBackground);

    await page.reload();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    expect(await background(page)).toBe(lightBackground);
    await page.getByRole("button", { name: "Menu" }).click();
    await expect(page.getByRole("radiogroup", { name: "Mode" }).getByLabel("Light")).toBeChecked();

    await chooseTheme(page, "Auto");
    await expect(page.locator("html")).not.toHaveAttribute("data-theme", /.*/);
    expect(await background(page)).toBe(darkBackground);
  });

  test("the stored choice applies before first paint", async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem("matchmark-theme", "light"));
    await page.goto("/");
    const early = await page.evaluate(() => document.documentElement.dataset.theme);
    expect(early).toBe("light");
  });

  test("Dark overrides a light OS", async ({ page }) => {
    await page.emulateMedia({ colorScheme: "light" });
    await page.goto("/");
    expect(await background(page)).toBe(lightBackground);
    await chooseTheme(page, "Dark");
    expect(await background(page)).toBe(darkBackground);
  });

  test("print stays light and the login logo follows the theme", async ({ page }) => {
    await page.goto("/admin");
    await expect(page.locator(".logo-dark")).toBeVisible();
    await page.evaluate(() => document.documentElement.setAttribute("data-theme", "light"));
    await expect(page.locator(".logo-light")).toBeVisible();
    await expect(page.locator(".logo-dark")).toBeHidden();
    await page.emulateMedia({ colorScheme: "dark", media: "print" });
    await page.evaluate(() => document.documentElement.setAttribute("data-theme", "dark"));
    expect(await background(page)).toBe(lightBackground);
  });

  test("the menu is keyboard accessible and closes on Escape", async ({ page }) => {
    await page.goto("/");
    const trigger = page.getByRole("button", { name: "Menu" });
    await expect(trigger).toHaveAttribute("aria-haspopup", "dialog");
    await trigger.focus();
    await page.keyboard.press("Enter");
    await expect(trigger).toHaveAttribute("aria-expanded", "true");
    await page.keyboard.press("Escape");
    await expect(trigger).toHaveAttribute("aria-expanded", "false");
    await expect(trigger).toBeFocused();
  });

  test("the signed-in admin menu carries the Mode group", async ({ adminPage: page }) => {
    await chooseTheme(page, "Light");
    expect(await background(page)).toBe(lightBackground);
    const menu = page.getByRole("group", { name: "Account menu" });
    await expect(menu.getByRole("heading", { name: "Account" })).toBeVisible();
    await page.goto("/admin");
    expect(await background(page)).toBe(lightBackground);
  });
});
