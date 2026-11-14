export function requiredError(value: string, what: string): string {
  return value.trim() === "" ? `Enter ${what}.` : "";
}

export function dateError(value: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value.trim());
  if (!match) return "Enter a date as YYYY-MM-DD.";
  const [year, month, day] = [Number(match[1]), Number(match[2]), Number(match[3])];
  const parsed = new Date(Date.UTC(year, month - 1, day));
  const real =
    parsed.getUTCFullYear() === year &&
    parsed.getUTCMonth() === month - 1 &&
    parsed.getUTCDate() === day;
  return real ? "" : "Enter a real calendar date.";
}

export function dateRangeError(start: string, end: string): string {
  if (dateError(start) || dateError(end)) return "";
  return end.trim() < start.trim() ? "End date must not be before the start date." : "";
}
