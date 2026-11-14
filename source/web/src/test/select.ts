import { screen } from "@testing-library/svelte";
import type { UserEvent } from "@testing-library/user-event";

export async function choose(user: UserEvent, name: string | RegExp, option: string | RegExp) {
  await user.click(screen.getByRole("combobox", { name }));
  await user.click(await screen.findByRole("option", { name: option }));
}

export async function optionLabels(user: UserEvent, name: string | RegExp): Promise<string[]> {
  await user.click(screen.getByRole("combobox", { name }));
  const labels = (await screen.findAllByRole("option")).map((o) => o.textContent?.trim() ?? "");
  await user.keyboard("{Escape}");
  return labels;
}
