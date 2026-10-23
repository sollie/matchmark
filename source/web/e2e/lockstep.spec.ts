import { createCompetition, createPairing, defaultCourse, expect, openCompetition, row, seedEntry, setCounts, setStatus, test } from "./helpers";

test.describe("lockstep stages", () => {
  test("the admin starts the next stage only after every lane accepted, then devices announce it", async ({ adminPage: page, admin, openDevice }) => {
    const competition = await createCompetition(admin);
    const first = await seedEntry(admin, competition.id, 1);
    const second = await seedEntry(admin, competition.id, 2);
    await setStatus(admin, competition.id, "active");
    const lane1 = await openDevice((await createPairing(admin, competition.id, 1)).token);
    const lane2 = await openDevice((await createPairing(admin, competition.id, 2)).token);

    await openCompetition(page, competition.id);
    await expect(page.getByText("· Stage 1")).toBeVisible();
    await page.getByRole("button", { name: "Start next stage" }).click();
    await expect(page.getByRole("alert")).toBeVisible();

    for (const device of [lane1, lane2]) {
      await device.getByLabel("10 count", { exact: true }).fill("12");
      await device.getByRole("button", { name: "Save", exact: true }).click();
      await expect(device.getByRole("status")).toHaveText("Saved");
      await device.getByRole("button", { name: "Lock stage", exact: true }).click();
    }
    for (const device of [lane1, lane2]) {
      await device.getByRole("button", { name: "Review result" }).click();
      await device.getByRole("button", { name: "Accept" }).click();
      await expect(device.getByRole("status")).toHaveText("Result accepted");
    }
    await expect(row(page, first.shooter.last_name).getByText("Accepted", { exact: true }).first()).toBeVisible();
    await expect(row(page, second.shooter.last_name).getByText("Accepted", { exact: true }).first()).toBeVisible();

    await page.getByRole("button", { name: "Start next stage" }).click();
    await expect(page.getByText("· Stage 2")).toBeVisible();
    for (const device of [lane1, lane2]) {
      await expect(device.getByText("The next stage has started")).toBeVisible();
      await device.getByRole("button", { name: "Continue" }).click();
      await expect(device.getByText(/Stage 2\./)).toBeVisible();
      await expect(device.getByRole("tab")).toHaveCount(0);
    }
  });

  test("the last acceptance finishes the competition and devices show the finished screen", async ({ adminPage: page, admin, openDevice }) => {
    const course = await defaultCourse(admin);
    const competition = await createCompetition(admin);
    const first = await seedEntry(admin, competition.id, 1);
    const second = await seedEntry(admin, competition.id, 2);
    await setStatus(admin, competition.id, "active");
    const lane1 = await openDevice((await createPairing(admin, competition.id, 1)).token);

    for (let stage = 1; stage <= course.stages; stage++) {
      if (stage > 1) {
        const response = await admin.post(`/api/competitions/${competition.id}/next-stage`);
        expect(response.ok()).toBe(true);
      }
      for (const entry of [first, second]) {
        await setCounts(admin, competition.id, entry.entryId, stage, { "10": course.stageShots[stage - 1] }, true);
      }
    }

    await expect(lane1.getByText("The competition is finished")).toBeVisible();
    await openCompetition(page, competition.id);
    await expect(page.getByText("finished", { exact: true }).first()).toBeVisible();
    await expect(page.getByRole("button", { name: "Start next stage" })).toHaveCount(0);
  });
});
