import {
  createMatch,
  createPairing,
  defaultCourse,
  expect,
  openMatch,
  row,
  seedEntry,
  setCounts,
  setStatus,
  test,
} from "./helpers";

test.describe("lockstep stages", () => {
  test("the admin starts the next stage only after every lane accepted, then devices announce it", async ({
    adminPage: page,
    admin,
    openDevice,
  }) => {
    const match = await createMatch(admin);
    const first = await seedEntry(admin, match.id, 1);
    const second = await seedEntry(admin, match.id, 2);
    await setStatus(admin, match.id, "active");
    const lane1 = await openDevice((await createPairing(admin, match.id, 1)).token);
    const lane2 = await openDevice((await createPairing(admin, match.id, 2)).token);

    await openMatch(page, match.id);
    await expect(page.getByText("· Stage 1")).toBeVisible();
    await expect(page.getByRole("button", { name: "Start next stage" })).toBeDisabled();

    for (const device of [lane1, lane2]) {
      await device.getByLabel("10 count", { exact: true }).fill("12");
      await device.getByRole("button", { name: "Save", exact: true }).click();
      await expect(device.locator("#save-status")).toHaveText("Saved ✓");
      await device.getByRole("button", { name: "Lock stage", exact: true }).click();
    }
    for (const device of [lane1, lane2]) {
      await device.getByRole("button", { name: "Review result" }).click();
      await device.getByRole("button", { name: "Accept" }).click();
      await expect(device.getByRole("status")).toHaveText("Result accepted");
    }
    await expect(
      row(page, first.shooter.last_name).getByText("Accepted", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      row(page, second.shooter.last_name).getByText("Accepted", { exact: true }).first(),
    ).toBeVisible();

    await page.getByRole("button", { name: "Start next stage" }).click();
    await expect(page.getByText("· Stage 2")).toBeVisible();
    for (const device of [lane1, lane2]) {
      await expect(device.getByText("The next stage has started")).toBeVisible();
      await device.getByRole("button", { name: "Continue" }).click();
      await expect(device.getByText(/Stage 2\./)).toBeVisible();
      await expect(device.getByRole("tab")).toHaveCount(0);
    }
  });

  test("the last acceptance finishes the match and a device lands on its lane's scorecard", async ({
    adminPage: page,
    admin,
    openDevice,
  }) => {
    const course = await defaultCourse(admin);
    const match = await createMatch(admin);
    const first = await seedEntry(admin, match.id, 1);
    const second = await seedEntry(admin, match.id, 2);
    await setStatus(admin, match.id, "active");
    const lane1 = await openDevice((await createPairing(admin, match.id, 1)).token);

    for (let stage = 1; stage <= course.stages; stage++) {
      if (stage > 1) {
        const response = await admin.post(`/api/matches/${match.id}/next-stage`);
        expect(response.ok()).toBe(true);
      }
      for (const entry of [first, second]) {
        await setCounts(
          admin,
          match.id,
          entry.entryId,
          stage,
          { "10": course.stageShots[stage - 1] },
          true,
        );
      }
    }

    await expect(lane1).toHaveURL(`/scorecard/${match.id}/1`);
    await expect(lane1.getByRole("link", { name: "Download PNG" })).toBeVisible();
    expect(await lane1.evaluate(() => localStorage.getItem("mm_device_token"))).toBeNull();
    await openMatch(page, match.id);
    await expect(page.getByText("finished", { exact: true }).first()).toBeVisible();
    await expect(page.getByRole("button", { name: "Start next stage" })).toHaveCount(0);
  });
});
