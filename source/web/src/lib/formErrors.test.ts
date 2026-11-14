import { describe, expect, it } from "vitest";
import { dateError, dateRangeError, requiredError } from "./formErrors";

describe("form errors", () => {
  it("requires a non-blank value", () => {
    expect(requiredError("  ", "a name")).toBe("Enter a name.");
    expect(requiredError("Spring", "a name")).toBe("");
  });

  it("accepts only real calendar dates", () => {
    expect(dateError("2026-03-01")).toBe("");
    expect(dateError("01/03/2026")).toBe("Enter a date as YYYY-MM-DD.");
    expect(dateError("2026-02-30")).toBe("Enter a real calendar date.");
  });

  it("rejects an end before the start", () => {
    expect(dateRangeError("2026-03-02", "2026-03-01")).toBe(
      "End date must not be before the start date.",
    );
    expect(dateRangeError("2026-03-01", "2026-03-01")).toBe("");
    expect(dateRangeError("nope", "2026-03-01")).toBe("");
  });
});
