import {
  expect,
  row,
  seedEntry,
  setStatus,
  test,
  createMatch,
  uniqueClientIp,
  uniqueName,
} from "./helpers";

test.describe("remote-controlled screens", () => {
  test("an admin approves a screen, drives it through a match and removes it", async ({
    adminPage: admin,
    admin: api,
    browser,
    baseURL,
  }) => {
    const draft = await createMatch(api, { name: uniqueName("Remote") });
    await seedEntry(api, draft.id, 1);
    await seedEntry(api, draft.id, 2);

    const context = await browser.newContext({
      baseURL,
      extraHTTPHeaders: { "X-Forwarded-For": uniqueClientIp() },
    });
    const viewer = await context.newPage();
    await viewer.goto("/");
    await viewer.getByRole("button", { name: "Enable remote control" }).click();
    const region = viewer.getByRole("region", { name: "Remote control" });
    const code = (await region.locator(".code").innerText()).trim();
    expect(code).toHaveLength(6);

    const screenName = uniqueName("Hall");
    await admin.goto("/admin/screens");
    await admin.getByLabel("Pending code").fill(code);
    await admin.getByLabel("Screen name").fill(screenName);
    await admin.getByRole("button", { name: "Approve screen" }).click();
    await expect(row(admin, screenName)).toBeVisible();

    await row(admin, screenName)
      .getByLabel(`Match for ${screenName}`)
      .selectOption({ label: `${draft.name} (${draft.date}, draft)` });
    await expect(viewer.getByRole("heading", { name: "Waiting for a match" })).toBeVisible();
    await expect(viewer.getByText(draft.name)).toHaveCount(0);

    await setStatus(api, draft.id, "active");
    await expect(viewer.getByRole("heading", { name: "Standings" })).toBeVisible();
    await expect(viewer.getByRole("heading", { name: "Waiting for a match" })).toHaveCount(0);

    const other = await createMatch(api, { name: uniqueName("Everywhere") });
    await seedEntry(api, other.id, 1);
    await admin.reload();
    await admin
      .getByLabel("Match for all screens")
      .selectOption({ label: `${other.name} (${other.date}, draft)` });
    await admin.getByRole("button", { name: "Apply to all screens" }).click();
    await expect(admin.getByRole("status")).toContainText("applied to all screens");
    await expect(viewer.getByRole("heading", { name: "Waiting for a match" })).toBeVisible();

    await setStatus(api, other.id, "active");
    await expect(viewer.getByRole("heading", { name: "Standings" })).toBeVisible();
    await setStatus(api, other.id, "finished");
    await expect(viewer.getByText("Finished").first()).toBeVisible();

    await viewer.getByRole("button", { name: "Leave remote control" }).click();
    await expect(viewer.getByRole("button", { name: "Enable remote control" })).toBeVisible();
    await expect(row(admin, screenName)).toHaveCount(0);

    await viewer.getByRole("button", { name: "Enable remote control" }).click();
    const second = (await region.locator(".code").innerText()).trim();
    const secondName = uniqueName("Lobby");
    await admin.getByLabel("Pending code").fill(second);
    await admin.getByLabel("Screen name").fill(secondName);
    await admin.getByRole("button", { name: "Approve screen" }).click();
    await expect(row(admin, secondName)).toBeVisible();
    await row(admin, secondName).getByRole("button", { name: "Remove" }).click();
    await expect(row(admin, secondName)).toHaveCount(0);
    await expect(viewer.getByRole("alert")).toBeVisible();
    await expect(viewer.getByRole("button", { name: "Enable remote control" })).toBeVisible();

    await context.close();
  });
});
