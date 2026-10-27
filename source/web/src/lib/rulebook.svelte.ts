import { api } from "./api/client";
import type { components } from "./api/schema";

export type Rulebook = components["schemas"]["Rulebook"];
export type RulebookCourse = components["schemas"]["RulebookCourse"];

export const rulebook = $state<{ courses: RulebookCourse[] }>({ courses: [] });

let loading: Promise<void> | null = null;

export function loadRulebook(): Promise<void> {
  loading ??= api.GET("/api/rulebook").then(({ data }) => {
    if (data) rulebook.courses = data.courses;
    else loading = null;
  });
  return loading;
}

export function courseLabel(key: string): string {
  return rulebook.courses.find((c) => c.key === key)?.label ?? key;
}

export function courseCategories(key: string): string[] {
  return rulebook.courses.find((c) => c.key === key)?.categories ?? [];
}

export function stageLabel(key: string, stageNumber: number): string {
  const part = rulebook.courses
    .find((c) => c.key === key)
    ?.stages?.find((s) => s.stage_number === stageNumber)?.part;
  return formatStageLabel(stageNumber, part);
}

export function formatStageLabel(stageNumber: number, part?: string): string {
  return part ? `Stage ${stageNumber} · ${part}` : `Stage ${stageNumber}`;
}
