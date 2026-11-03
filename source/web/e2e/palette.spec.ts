import { expect, openAccountLink, test } from "./helpers";

test.describe("club palette", () => {
  test("an admin picks a palette and a spectator page keeps it across reloads", async ({
    adminPage: page,
    admin,
    browser,
    baseURL,
  }) => {
    try {
      await openAccountLink(page, "Club");
      const picker = page.getByRole("combobox", { name: "Palette" });
      await expect(picker).toHaveValue("matchmark");
      await expect(page.locator("html")).toHaveAttribute("data-palette", "matchmark");

      await picker.selectOption("arctic-blue");
      await expect(page.locator("html")).toHaveAttribute("data-palette", "arctic-blue");
      await expect(page.getByRole("status")).toHaveText("Palette saved.");

      const spectator = await browser.newContext({ baseURL });
      const view = await spectator.newPage();
      await view.goto("/");
      await expect(view.locator("html")).toHaveAttribute("data-palette", "arctic-blue");
      await view.reload();
      await expect(view.locator("html")).toHaveAttribute("data-palette", "arctic-blue");
      await spectator.close();
    } finally {
      await admin.put("/api/club", { data: { palette: "matchmark" } });
    }
  });
});
