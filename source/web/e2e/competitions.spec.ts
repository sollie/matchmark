import { expect, test, uniqueName } from "./helpers";

test.describe("competitions", () => {
  test("create a competition, group matches under it and enforce its date range", async ({ adminPage: page }) => {
    const competition = uniqueName("Nationals");
    const match = uniqueName("Day two");

    await page.getByRole("button", { name: "New competition" }).click();
    const form = page.getByRole("form", { name: "New competition" });
    await form.getByLabel("Competition name").fill(competition);
    await form.getByLabel("Start date").fill("2026-10-19");
    await form.getByLabel("End date").fill("2026-10-20");
    await form.getByRole("button", { name: "Save" }).click();
    const group = page.getByRole("region", { name: `Competition ${competition}` });
    await expect(group).toContainText("2026-10-19 – 2026-10-20");

    await page.getByLabel("Name", { exact: true }).fill(match);
    await page.getByLabel("Date", { exact: true }).fill("2026-10-21");
    await page.getByRole("combobox", { name: /^Competition/ }).selectOption({ label: `${competition} (2026-10-19 – 2026-10-20)` });
    await page.getByRole("button", { name: "Create match" }).click();
    await expect(page.getByRole("alert")).toBeVisible();

    await page.getByLabel("Date", { exact: true }).fill("2026-10-20");
    await page.getByRole("button", { name: "Create match" }).click();
    await expect(page.getByRole("heading", { name: match })).toBeVisible();

    await page.goto("/admin");
    await expect(group.getByRole("row").filter({ hasText: match })).toHaveCount(1);
    await expect(group.getByRole("button", { name: /^Delete competition/ })).toHaveCount(0);

    await group.getByRole("button", { name: `Edit competition ${competition}` }).click();
    const edit = page.getByRole("form", { name: "Edit competition" });
    await edit.getByLabel("End date").fill("2026-10-19");
    await edit.getByRole("button", { name: "Save" }).click();
    await expect(edit.getByRole("alert")).toBeVisible();
    await edit.getByLabel("End date").fill("2026-10-22");
    await edit.getByRole("button", { name: "Save" }).click();
    await expect(group).toContainText("2026-10-19 – 2026-10-22");
  });

  test("an empty competition can be deleted", async ({ adminPage: page }) => {
    const competition = uniqueName("Empty");

    await page.getByRole("button", { name: "New competition" }).click();
    const form = page.getByRole("form", { name: "New competition" });
    await form.getByLabel("Competition name").fill(competition);
    await form.getByRole("button", { name: "Save" }).click();
    const group = page.getByRole("region", { name: `Competition ${competition}` });
    await expect(group).toBeVisible();

    page.once("dialog", (dialog) => dialog.accept());
    await group.getByRole("button", { name: `Delete competition ${competition}` }).click();
    await expect(group).toHaveCount(0);
  });

  test("a new competition is created together with the match", async ({ adminPage: page }) => {
    const competition = uniqueName("Combined");
    const match = uniqueName("Opening");

    await page.getByLabel("Name", { exact: true }).fill(match);
    await page.getByLabel("Date", { exact: true }).fill("2026-11-02");
    await page.getByRole("combobox", { name: /^Competition/ }).selectOption({ label: "New competition…" });
    await page.getByLabel("Competition name").fill(competition);
    await page.getByLabel("Start date").fill("2026-11-01");
    await page.getByLabel("End date").fill("2026-11-03");
    await page.getByRole("button", { name: "Create match" }).click();
    await expect(page.getByRole("heading", { name: match })).toBeVisible();

    await page.goto("/admin");
    const group = page.getByRole("region", { name: `Competition ${competition}` });
    await expect(group.getByRole("row").filter({ hasText: match })).toHaveCount(1);
  });
});
