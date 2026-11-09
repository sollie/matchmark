import { createMatch, expect, openMatch, test, uniqueName } from "./helpers";

test.describe("competitions", () => {
  test("create a competition, group matches under it and enforce its date span", async ({
    adminPage: page,
  }) => {
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
    await page
      .getByRole("combobox", { name: /^Competition/ })
      .selectOption({ label: `${competition} (2026-10-19 – 2026-10-20)` });
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
    await page
      .getByRole("combobox", { name: /^Competition/ })
      .selectOption({ label: "New competition…" });
    await page.getByLabel("Competition name").fill(competition);
    await page.getByLabel("Start date").fill("2026-11-01");
    await page.getByLabel("End date").fill("2026-11-03");
    await page.getByRole("button", { name: "Create match" }).click();
    await expect(page.getByRole("heading", { name: match })).toBeVisible();

    await page.goto("/admin");
    const group = page.getByRole("region", { name: `Competition ${competition}` });
    await expect(group.getByRole("row").filter({ hasText: match })).toHaveCount(1);
  });

  test("the competition form sends several officials in order and shows a failed lookup", async ({
    adminPage: page,
  }) => {
    let body: Record<string, unknown> = {};
    await page.route("**/api/competitions", async (route) => {
      if (route.request().method() !== "POST") return route.fallback();
      body = route.request().postDataJSON();
      await route.fulfill({
        status: 422,
        json: { error: "WAID not found in the WA1500 register" },
      });
    });

    await page.getByRole("button", { name: "New competition" }).click();
    const form = page.getByRole("form", { name: "New competition" });
    await form.getByLabel("Competition name").fill(uniqueName("Officials"));
    await form.getByRole("button", { name: "Add referee" }).click();
    await form.getByLabel("Referee WAID 1").fill("2002");
    await form.getByRole("button", { name: "Add referee" }).click();
    await form.getByLabel("Referee WAID 2").fill("1001");
    await form.getByRole("button", { name: "Add range officer" }).click();
    await form.getByLabel("Range officer WAID 1").fill("3003");
    await form.getByRole("button", { name: "Save" }).click();

    await expect(form.getByRole("alert")).toHaveText("WAID not found in the WA1500 register");
    expect(body).toMatchObject({
      referee_waids: ["2002", "1001"],
      range_officer_waids: ["3003"],
    });
  });

  test("a match shows the officials it inherits and its edit form leaves its own empty", async ({
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    await page.route(`**/api/matches/${match.id}`, async (route) => {
      if (route.request().method() !== "GET") return route.fallback();
      const real = await route.fetch();
      const json = await real.json();
      await route.fulfill({
        response: real,
        json: {
          ...json,
          effective_referees: [{ waid: "1001", name: "Rita Referee" }],
          effective_range_officers: [
            { waid: "2002", name: "Ron Range" },
            { waid: "3003", name: "Ola Officer" },
          ],
        },
      });
    });

    await openMatch(page, match.id);
    await expect(page.getByText("Referee: Rita Referee (1001)")).toBeVisible();
    await expect(
      page.getByText("Range officers: Ron Range (2002), Ola Officer (3003)"),
    ).toBeVisible();

    await page.getByRole("button", { name: "Edit match" }).click();
    const edit = page.getByRole("form", { name: "Edit match" });
    await expect(edit.getByLabel(/^Referee WAID/)).toHaveCount(0);
    await expect(
      edit.getByText("Leave a role empty to use the competition's officials."),
    ).toBeVisible();
  });

  test("officials are picked from users with a WAID and the signed-in user is the default Referee", async ({
    adminPage: page,
    admin,
  }) => {
    const username = uniqueName("official").replace(" ", "-");
    const ownWaid = String(Math.floor(1e8 + Math.random() * 9e8));
    const otherWaid = String(Math.floor(1e8 + Math.random() * 9e8));
    const created = await admin.post("/api/users", {
      data: { username, role: "user", password: "e2e-temporary-password-1" },
    });
    const { id } = await created.json();
    await admin.put(`/api/users/${id}/waid`, { data: { waid: otherWaid } });
    await admin.put("/api/admin/waid", { data: { waid: ownWaid } });

    try {
      await page.goto("/admin");
      let body: Record<string, unknown> = {};
      await page.route("**/api/competitions", async (route) => {
        if (route.request().method() !== "POST") return route.fallback();
        body = route.request().postDataJSON();
        await route.fulfill({ status: 422, json: { error: "stubbed" } });
      });

      await page.getByRole("button", { name: "New competition" }).click();
      const form = page.getByRole("form", { name: "New competition" });
      await expect(form.getByLabel("Referee WAID 1")).toHaveValue(ownWaid);
      await expect(form.getByLabel("Pick referee from users")).not.toContainText(`(${ownWaid})`);
      await form
        .getByLabel("Pick referee from users")
        .selectOption({ label: `${username} (${otherWaid})` });
      await expect(form.getByLabel("Referee WAID 2")).toHaveValue(otherWaid);
      await form
        .getByLabel("Pick range officer from users")
        .selectOption({ label: `${username} (${otherWaid})` });
      await expect(form.getByLabel("Range officer WAID 1")).toHaveValue(otherWaid);

      await form.getByLabel("Competition name").fill(uniqueName("Picked"));
      await form.getByRole("button", { name: "Save" }).click();
      await expect(form.getByRole("alert")).toHaveText("stubbed");
      expect(body).toMatchObject({
        referee_waids: [ownWaid, otherWaid],
        range_officer_waids: [otherWaid],
      });
    } finally {
      await admin.put("/api/admin/waid", { data: { waid: "" } });
    }
  });
});
