import { expect, openAccountLink, test, uniqueName } from "./helpers";

test.describe("ranges", () => {
  test("the default range pre-fills a new competition and a used range can only be archived", async ({
    adminPage: page,
  }) => {
    const range = uniqueName("Outdoor");
    const competition = uniqueName("Range cup");

    await openAccountLink(page, "Club");
    await page.getByRole("textbox", { name: "Range to add" }).fill(range);
    await page.getByRole("button", { name: "Add range" }).click();
    await expect(page.getByRole("status")).toHaveText("Range added.");
    await page.getByLabel("Default range").selectOption({ label: range });
    await expect(page.getByRole("status")).toHaveText("Default range saved.");

    const row = page.getByRole("listitem", { name: `Range ${range}` });
    await expect(row).toContainText("(default)");
    await expect(row.getByRole("button", { name: `Archive range ${range}` })).toHaveCount(0);

    await page.goto("/admin");
    await page.getByRole("button", { name: "New competition" }).click();
    const form = page.getByRole("form", { name: "New competition" });
    await expect(form.getByLabel(/^Range(?! officer)/)).toHaveText(new RegExp(range));
    await form.getByLabel("Competition name").fill(competition);
    await form.getByRole("button", { name: "Save" }).click();
    const group = page.getByRole("region", { name: `Competition ${competition}` });
    await expect(group).toContainText(range);

    await openAccountLink(page, "Club");
    await page.getByLabel("Default range").selectOption({ label: "None" });
    await expect(page.getByRole("status")).toHaveText("Default range saved.");
    page.once("dialog", (dialog) => dialog.accept());
    await row.getByRole("button", { name: `Delete range ${range}` }).click();
    await expect(page.getByRole("alert")).toBeVisible();
    await row.getByRole("button", { name: `Archive range ${range}` }).click();
    await expect(row).toContainText("(archived)");
    await expect(page.getByLabel("Default range")).not.toHaveText(new RegExp(range));
  });
});
