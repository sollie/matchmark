import { expect, test } from "./helpers";

const pixelPng = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg==",
  "base64",
);

test.describe("club settings", () => {
  test("an admin sets the club name and logo, which then show on the public page", async ({
    adminPage: page,
  }) => {
    await page.getByRole("link", { name: "Club", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Club settings" })).toBeVisible();

    await page.getByRole("textbox", { name: "Name" }).fill("E2E Pistol Club");
    await page.getByRole("button", { name: "Save name" }).click();
    await expect(page.getByRole("status")).toHaveText("Club name saved.");

    await page
      .getByLabel("Logo file")
      .setInputFiles({ name: "logo.png", mimeType: "image/png", buffer: pixelPng });
    await page.getByRole("button", { name: "Upload logo" }).click();
    await expect(page.getByRole("status")).toHaveText("Logo saved.");
    await expect(page.getByRole("img", { name: "Current club logo" })).toBeVisible();

    await page.getByLabel("Logo file").setInputFiles({
      name: "logo.svg",
      mimeType: "image/svg+xml",
      buffer: Buffer.from("<svg xmlns='http://www.w3.org/2000/svg'/>"),
    });
    await page.getByRole("button", { name: "Upload logo" }).click();
    await expect(page.getByRole("alert")).toContainText("PNG or JPEG");

    await page.goto("/");
    await expect(page.locator(".club-badge")).toContainText("E2E Pistol Club");
    await expect(page.locator(".club-badge img")).toHaveAttribute(
      "src",
      /\/api\/club\/logo\?v=\d+/,
    );

    await page.goto("/admin/club");
    await page.getByRole("button", { name: "Remove logo" }).click();
    await expect(page.getByText("No logo uploaded.")).toBeVisible();
    await page.getByRole("textbox", { name: "Name" }).fill("");
    await page.getByRole("button", { name: "Save name" }).click();
    await expect(page.getByRole("status")).toHaveText("Club name saved.");
  });
});
